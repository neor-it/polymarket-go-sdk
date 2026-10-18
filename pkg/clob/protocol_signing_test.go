package clob

import (
	"context"
	"encoding/json"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/positions"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"github.com/shopspring/decimal"
)

type recordingOrderSigner struct {
	auth.Signer
	domain *apitypes.TypedDataDomain
}

func (s *recordingOrderSigner) SignTypedData(domain *apitypes.TypedDataDomain, definitions apitypes.Types, message apitypes.TypedDataMessage, primaryType string) ([]byte, error) {
	copied := *domain
	s.domain = &copied
	return s.Signer.SignTypedData(domain, definitions, message, primaryType)
}

func TestExplicitProtocolSelectsEverySigningPath(t *testing.T) {
	signer := mustSigner(t)
	wallet := common.HexToAddress("0x123")
	for _, signatureType := range []int{int(auth.SignatureEOA), int(auth.SignatureGnosisSafe)} {
		recorder := &recordingOrderSigner{Signer: signer}
		order := newPoly1271Order(wallet, "BUY")
		order.ProtocolVersion = types.ProtocolV2
		order.SignatureType = &signatureType
		order.Signer = signer.Address()
		if signatureType == int(auth.SignatureEOA) {
			order.Maker = signer.Address()
		}
		_, err := SignOrder(recorder, &auth.APIKey{Key: "fixture"}, order)
		if err != nil {
			t.Fatal(err)
		}
		if recorder.domain == nil || recorder.domain.Version != "3" || !strings.EqualFold(recorder.domain.VerifyingContract, positions.ExchangeV3Address) {
			t.Fatalf("wrong V2 domain %+v", recorder.domain)
		}
		order.ProtocolVersion = types.ProtocolV1
		_, err = SignOrder(recorder, &auth.APIKey{Key: "fixture"}, order)
		if err != nil {
			t.Fatal(err)
		}
		if recorder.domain.Version != "2" || !strings.EqualFold(recorder.domain.VerifyingContract, exchangeV2Address) {
			t.Fatal("CTF signing changed")
		}
	}
	order := newPoly1271Order(wallet, "SELL")
	order.ProtocolVersion = types.ProtocolV2
	signed, err := SignOrder(signer, &auth.APIKey{Key: "fixture"}, order)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := BuildPoly1271OrderEnvelope(order, positions.PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	var typed apitypes.TypedData
	if err := json.Unmarshal(envelope.TypedData, &typed); err != nil {
		t.Fatal(err)
	}
	if typed.Domain.Version != "3" || typed.Message["version"] != "1" {
		t.Fatalf("exchange/deposit wallet domains confused %+v", typed)
	}
	signature, err := signer.SignTypedData(&typed.Domain, typed.Types, typed.Message, typed.PrimaryType)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := AssembleTypedDataSignSignature(hexutil.Encode(signature), envelope)
	if err != nil {
		t.Fatal(err)
	}
	if signed.Signature != wrapped {
		t.Fatal("internal and external V2 POLY_1271 signatures disagree")
	}
	order.ProtocolVersion = types.ProtocolV1
	legacy, err := BuildPoly1271OrderEnvelope(order, positions.PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.AppDomainSeparator == envelope.AppDomainSeparator {
		t.Fatal("CTF and PositionManager share signing identity")
	}
	payload, err := buildOrderPayload(signed)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), "protocol_version") {
		t.Fatal("local protocol metadata leaked into CLOB wire")
	}
}

func TestBuilderPropagatesProtocolForLimitAndMarketOrders(t *testing.T) {
	for _, market := range []bool{false, true} {
		stub := newStubClient()
		stub.tickSize = 0.01
		stub.feeRate = 0
		builder := NewOrderBuilder(stub, mustSigner(t)).ProtocolVersion(types.ProtocolV2).TokenID("123").Side("BUY").Price(0.5)
		var order *clobtypes.Order
		var err error
		if market {
			result, buildErr := builder.AmountUSDC(5).OrderType(clobtypes.OrderTypeFOK).BuildMarket()
			err = buildErr
			if result != nil {
				order = result.Order
			}
		} else {
			order, err = builder.Size(10).Build()
		}
		if err != nil || order == nil || order.ProtocolVersion != types.ProtocolV2 {
			t.Fatalf("protocol lost %+v %v", order, err)
		}
	}
	for _, protocol := range []types.ProtocolVersion{"future", types.ProtocolV2} {
		chain := int64(positions.PolygonChainID)
		if protocol == types.ProtocolV2 {
			chain = 1
		}
		if _, err := SigningConfiguration(chain, protocol, false); err == nil {
			t.Fatal("unsupported protocol/chain accepted")
		}
	}
}

func TestIntegerFillFloorAndSeparateFees(t *testing.T) {
	order := &clobtypes.Order{TokenID: types.U256{Int: big.NewInt(123)}, MakerAmount: decimal.NewFromInt(3), TakerAmount: decimal.NewFromInt(10), Side: "BUY"}
	fill, err := CalculateFill(order, big.NewInt(2), big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if fill.Shares.Int64() != 6 || fill.CollateralDebit.Int64() != 3 || fill.Fee.Int64() != 1 {
		t.Fatalf("BUY rounding %+v", fill)
	}
	order.Side = "SELL"
	fill, err = CalculateFill(order, big.NewInt(2), big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if fill.Shares.Int64() != 2 || fill.CollateralCredit.Int64() != 5 || fill.TakerAsset.Int64() != 6 {
		t.Fatalf("SELL rounding %+v", fill)
	}
	if _, err := CalculateFill(order, big.NewInt(4), big.NewInt(0)); err == nil {
		t.Fatal("overfill accepted")
	}
	if _, err := CalculateFill(order, big.NewInt(1), big.NewInt(4)); err == nil {
		t.Fatal("sell fee exceeds proceeds")
	}
	order.MakerAmount = decimal.RequireFromString("3.5")
	if _, err := CalculateFill(order, big.NewInt(1), big.NewInt(0)); err == nil {
		t.Fatal("fractional base units silently truncated")
	}
}

func TestV2AllowanceRefreshUsesExplicitLedgerAndMakerSignature(t *testing.T) {
	const token = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("asset_type") != "CONDITIONAL-V2" || q.Get("token_id") != token || q.Get("signature_type") != "3" {
			t.Fatalf("wrong position ledger refresh %s", r.URL.RawQuery)
		}
		if r.URL.Path != "/balance-allowance" && r.URL.Path != "/balance-allowance/update" {
			t.Fatalf("wrong balance path %s", r.URL.Path)
		}
		if _, err := w.Write([]byte(`{"balance":"1000000","allowances":{"` + positions.ExchangeV3Address + `":"1000000","` + exchangeV2Address + `":"0"}}`)); err != nil {
			t.Errorf("write test response: %v", err)
		}
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	signature := int(auth.SignaturePoly1271)
	_, err := client.UpdateBalanceAllowance(context.Background(), &clobtypes.BalanceAllowanceUpdateRequest{AssetType: clobtypes.AssetTypeConditionalV2, TokenID: token, SignatureType: &signature})
	if err != nil {
		t.Fatal(err)
	}
	read, err := client.BalanceAllowance(context.Background(), &clobtypes.BalanceAllowanceRequest{AssetType: clobtypes.AssetTypeConditionalV2, TokenID: token, SignatureType: &signature})
	if err != nil {
		t.Fatal(err)
	}
	if read.Allowances[positions.ExchangeV3Address] != "1000000" || read.Allowances[exchangeV2Address] != "0" {
		t.Fatalf("spender permissions collapsed %+v", read)
	}
}

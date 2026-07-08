package relayer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

// newRelayerClientForServer builds a signer-less relayer Client pointed at a
// test server. ResolveDepositWallet only calls IsWalletDeployed, which needs
// no signer.
func newRelayerClientForServer(t *testing.T, url string) Client {
	t.Helper()
	return NewClient(transport.NewClient(nil, url))
}

// Addresses verified two ways: against @polymarket/builder-relayer-client's
// derive.js (dist/builder/derive.js, deriveUupsDepositWallet /
// deriveBeaconDepositWallet) and against live Polygon mainnet bytecode at each
// address (eth_getCode).
func TestDeriveDepositWalletAddresses(t *testing.T) {
	factory := common.HexToAddress(FactoryPolygon)
	beacon := common.HexToAddress(BeaconPolygon)
	implementation := common.HexToAddress(ImplementationPolygon)

	cases := []struct {
		name       string
		owner      string
		wantUups   string
		wantBeacon string
	}{
		{
			name:       "owner with a legacy UUPS wallet on-chain",
			owner:      "0x888Ac92C7E7784e55A843ea38375881e9909E520",
			wantUups:   "0xE2043299A1BBEf5F36267DB6008170Dd7191df01",
			wantBeacon: "0xA8a0DaF07897Ce796BB5538D4aCDB107D891976B",
		},
		{
			name:       "owner with a current BeaconProxy wallet on-chain",
			owner:      "0xaB2b3380c0a48587Da256D83CEebC5Dbe0651fdd",
			wantUups:   "0x173c4F55b8b2B9F492128B078B3F84BAE4fd9660",
			wantBeacon: "0x57b74C70eC11d74304cC144c0adf74503eeeB426",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner := common.HexToAddress(tc.owner)

			gotUups := DeriveUupsDepositWallet(owner, factory, implementation)
			if !strings.EqualFold(gotUups.Hex(), tc.wantUups) {
				t.Errorf("DeriveUupsDepositWallet(%s) = %s, want %s", tc.owner, gotUups.Hex(), tc.wantUups)
			}

			gotBeacon := DeriveBeaconDepositWallet(owner, factory, beacon)
			if !strings.EqualFold(gotBeacon.Hex(), tc.wantBeacon) {
				t.Errorf("DeriveBeaconDepositWallet(%s) = %s, want %s", tc.owner, gotBeacon.Hex(), tc.wantBeacon)
			}
		})
	}
}

// deployedOnlyServer returns an httptest.Server whose /deployed handler
// reports exactly one address (case-insensitive) as deployed.
func deployedOnlyServer(t *testing.T, deployedAddress common.Address) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deployed" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		queried := r.URL.Query().Get("address")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{
			"deployed": strings.EqualFold(queried, deployedAddress.Hex()),
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestResolveDepositWalletPrefersBeaconShape(t *testing.T) {
	owner := common.HexToAddress("0xaB2b3380c0a48587Da256D83CEebC5Dbe0651fdd")
	wantBeacon := DeriveBeaconDepositWallet(owner, common.HexToAddress(FactoryPolygon), common.HexToAddress(BeaconPolygon))

	srv := deployedOnlyServer(t, wantBeacon)
	c := newRelayerClientForServer(t, srv.URL)

	got, deployed, err := ResolveDepositWallet(context.Background(), c, owner)
	if err != nil {
		t.Fatalf("ResolveDepositWallet: %v", err)
	}
	if !deployed {
		t.Fatalf("expected deployed=true")
	}
	if got != wantBeacon {
		t.Fatalf("resolved wallet = %s, want beacon-shape address %s", got.Hex(), wantBeacon.Hex())
	}
}

func TestResolveDepositWalletFallsBackToUupsShape(t *testing.T) {
	owner := common.HexToAddress("0x888Ac92C7E7784e55A843ea38375881e9909E520")
	wantUups := DeriveUupsDepositWallet(owner, common.HexToAddress(FactoryPolygon), common.HexToAddress(ImplementationPolygon))

	srv := deployedOnlyServer(t, wantUups)
	c := newRelayerClientForServer(t, srv.URL)

	got, deployed, err := ResolveDepositWallet(context.Background(), c, owner)
	if err != nil {
		t.Fatalf("ResolveDepositWallet: %v", err)
	}
	if !deployed {
		t.Fatalf("expected deployed=true")
	}
	if got != wantUups {
		t.Fatalf("resolved wallet = %s, want UUPS-shape address %s", got.Hex(), wantUups.Hex())
	}
}

func TestResolveDepositWalletReportsNotDeployed(t *testing.T) {
	owner := common.HexToAddress("0x0000000000000000000000000000000000dEaD")
	srv := deployedOnlyServer(t, common.Address{}) // nothing matches
	c := newRelayerClientForServer(t, srv.URL)

	wantBeacon := DeriveBeaconDepositWallet(owner, common.HexToAddress(FactoryPolygon), common.HexToAddress(BeaconPolygon))

	got, deployed, err := ResolveDepositWallet(context.Background(), c, owner)
	if err != nil {
		t.Fatalf("ResolveDepositWallet: %v", err)
	}
	if deployed {
		t.Fatalf("expected deployed=false")
	}
	if got != wantBeacon {
		t.Fatalf("not-deployed fallback address = %s, want beacon-shape address %s", got.Hex(), wantBeacon.Hex())
	}
}

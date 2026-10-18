package clob

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

func buildOrderPayload(order *clobtypes.SignedOrder) (map[string]interface{}, error) {
	if order == nil {
		return nil, fmt.Errorf("order is required")
	}
	orderType := normalizeOrderType(order.OrderType, clobtypes.OrderTypeGTC)
	if order.PostOnly != nil && *order.PostOnly && orderType != clobtypes.OrderTypeGTC && orderType != clobtypes.OrderTypeGTD {
		return nil, fmt.Errorf("postOnly is only supported for GTC and GTD orders")
	}
	orderMap, err := orderWithSignature(order)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"order":     orderMap,
		"owner":     order.Owner,
		"orderType": orderType,
	}
	if order.PostOnly != nil {
		payload["postOnly"] = *order.PostOnly
	}
	if order.DeferExec != nil {
		payload["deferExec"] = *order.DeferExec
	} else if isPoly1271SignedOrder(order) {
		payload["deferExec"] = false
	}
	return payload, nil
}

func isPoly1271SignedOrder(order *clobtypes.SignedOrder) bool {
	return order != nil && order.Order.SignatureType != nil && *order.Order.SignatureType == 3
}

func buildOrdersPayload(orders *clobtypes.SignedOrders) ([]map[string]interface{}, error) {
	if orders == nil {
		return nil, fmt.Errorf("orders are required")
	}
	payloads := make([]map[string]interface{}, 0, len(orders.Orders))
	for idx := range orders.Orders {
		order := orders.Orders[idx]
		payload, err := buildOrderPayload(&order)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func orderWithSignature(order *clobtypes.SignedOrder) (map[string]interface{}, error) {
	if order == nil {
		return nil, fmt.Errorf("order is required")
	}
	if order.Signature == "" {
		return nil, fmt.Errorf("signature is required")
	}
	if order.Owner == "" {
		return nil, fmt.Errorf("owner is required")
	}

	sigType := 0
	if order.Order.SignatureType != nil {
		sigType = *order.Order.SignatureType
	}

	side := strings.ToUpper(order.Order.Side)
	if side != "BUY" && side != "SELL" {
		return nil, fmt.Errorf("invalid order side %q", order.Order.Side)
	}

	salt, err := saltToJSON(order.Order.Salt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"salt":          salt,
		"maker":         order.Order.Maker.Hex(),
		"signer":        order.Order.Signer.Hex(),
		"tokenId":       u256String(order.Order.TokenID),
		"makerAmount":   decimalString(order.Order.MakerAmount),
		"takerAmount":   decimalString(order.Order.TakerAmount),
		"side":          side,
		"expiration":    u256String(order.Order.Expiration),
		"signatureType": sigType,
		"timestamp":     strconv.FormatInt(order.Order.Timestamp, 10),
		"metadata":      order.Order.Metadata.Hex(),
		"builder":       order.Order.Builder.Hex(),
		"signature":     order.Signature,
	}, nil
}

func u256String(value types.U256) string {
	if value.Int == nil {
		return "0"
	}
	return value.String()
}

func decimalString(value types.Decimal) string {
	return value.String()
}

func saltToJSON(value types.U256) (json.Number, error) {
	if value.Int == nil {
		return json.Number("0"), nil
	}
	if value.Sign() < 0 || value.BitLen() > orderUint256Bits {
		return "", fmt.Errorf("order salt must be a uint256")
	}
	return json.Number(value.String()), nil
}

func normalizeOrderType(orderType clobtypes.OrderType, fallback clobtypes.OrderType) clobtypes.OrderType {
	trimmed := strings.TrimSpace(string(orderType))
	if trimmed == "" {
		return fallback
	}
	upper := strings.ToUpper(trimmed)
	return clobtypes.OrderType(upper)
}

package bbTypes

import (
	"fmt"
	"math/big"
	"strings"
)

func ConvertScientificToDecimal(scientific string) (string, error) {
	if !strings.Contains(scientific, "e+") {
		return scientific, nil
	}

	parts := strings.Split(scientific, "e+")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid scientific notation format")
	}

	significand := parts[0]
	exponent := int64(ToInt(parts[1]))

	decimalValue := new(big.Float)
	decimalValue.SetString(significand)

	base10 := new(big.Int).SetInt64(10)
	expMultiplier := new(big.Float).SetInt(
		new(big.Int).Exp(
			base10,
			new(big.Int).SetInt64(exponent), nil,
		),
	)

	decimalValue.Mul(decimalValue, expMultiplier)

	return decimalValue.Text('f', 0), nil
}

func ToInt(s string) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0
	}
	return result
}

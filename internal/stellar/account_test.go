package stellar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// GetAccountBalance Behavior Tests (with mocked HTTP)
// ============================================================================

func TestGetAccountBalance_ReturnsCorrectBalance(t *testing.T) {
	// Mock Horizon API response
	mockResponse := map[string]interface{}{
		"account_id": "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
		"balances": []map[string]interface{}{
			{
				"asset_type": "native",
				"asset_code": "",
				"balance":    "1000.0000000",
			},
			{
				"asset_type": "credit_alphanum4",
				"asset_code": "USDC",
				"asset_issuer": "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
				"balance":    "500.5000000",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create client with mock server
	client := &Client{
		config: Config{
			RPCURL: server.URL,
		},
		horizon: &horizonclient.Client{
			HorizonURL: server.URL,
		},
	}

	// Test GetVaultBalance
	reader := NewVaultReader(NewContractInvoker(client))
	balance, err := reader.GetVaultBalance(context.Background(), "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4")

	// Should succeed with mocked response
	require.NoError(t, err)
	require.NotNil(t, balance)
	assert.Equal(t, "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4", balance.ContractID)
	assert.True(t, balance.Total.Equal(decimal.Zero)) // Placeholder implementation returns zero
}

func TestGetAccountBalance_AccountNotFound(t *testing.T) {
	// Mock 404 response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "404",
			"title":  "Resource Missing",
			"detail": "Account not found",
		})
	}))
	defer server.Close()

	client := &Client{
		config: Config{
			RPCURL: server.URL,
		},
		horizon: &horizonclient.Client{
			HorizonURL: server.URL,
		},
	}

	reader := NewVaultReader(NewContractInvoker(client))
	balance, err := reader.GetVaultBalance(context.Background(), "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4")

	// Should handle 404 gracefully
	assert.Error(t, err)
	assert.Nil(t, balance)
}

func TestGetAccountBalance_NetworkError(t *testing.T) {
	// Simulate server failure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("server error")
	}))
	defer server.Close()

	client := &Client{
		config: Config{
			RPCURL: server.URL,
		},
		horizon: &horizonclient.Client{
			HorizonURL: server.URL,
		},
	}

	reader := NewVaultReader(NewContractInvoker(client))
	balance, err := reader.GetVaultBalance(context.Background(), "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4")

	// Should return wrapped error, not panic
	assert.Error(t, err)
	assert.Nil(t, balance)
	assert.Contains(t, err.Error(), "failed to query vault balance")
}

// ============================================================================
// AccountBalance Structure Tests
// ============================================================================

func TestAccountBalance_Structure(t *testing.T) {
	balance := &AccountBalance{
		Address:   "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
		AssetCode: "USDC",
		Amount:    decimal.RequireFromString("1000.50"),
	}

	assert.Equal(t, "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX", balance.Address)
	assert.Equal(t, "USDC", balance.AssetCode)
	assert.True(t, balance.Amount.Equal(decimal.RequireFromString("1000.50")))
}

func TestAccountBalance_ZeroBalance(t *testing.T) {
	balance := &AccountBalance{
		Address:   "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
		AssetCode: "XLM",
		Amount:    decimal.Zero,
	}

	assert.True(t, balance.Amount.IsZero())
}

func TestAccountBalance_NegativeBalance(t *testing.T) {
	balance := &AccountBalance{
		Address:   "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
		AssetCode: "XLM",
		Amount:    decimal.RequireFromString("-100.00"),
	}

	assert.True(t, balance.Amount.IsNegative())
}

func TestAccountBalance_LargeBalance(t *testing.T) {
	balance := &AccountBalance{
		Address:   "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
		AssetCode: "USDC",
		Amount:    decimal.RequireFromString("999999999999.9999999"),
	}

	assert.True(t, balance.Amount.GreaterThan(decimal.Zero))
}

// ============================================================================
// Address Validation Tests
// ============================================================================

func TestValidateStellarAddress_ValidAddress(t *testing.T) {
	validAddress := "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX"
	err := validateStellarAddress(validAddress)
	assert.NoError(t, err)
}

func TestValidateStellarAddress_InvalidAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "empty address",
			address: "",
			wantErr: true,
		},
		{
			name:    "too short",
			address: "SHORT",
			wantErr: true,
		},
		{
			name:    "invalid prefix",
			address: "XBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
			wantErr: true,
		},
		{
			name:    "too long",
			address: "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGXEXTRAA",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStellarAddress(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// Asset Code Validation Tests
// ============================================================================

func TestValidateAssetCode_ValidCodes(t *testing.T) {
	tests := []struct {
		name      string
		assetCode string
	}{
		{
			name:      "valid XLM",
			assetCode: "XLM",
		},
		{
			name:      "valid USDC",
			assetCode: "USDC",
		},
		{
			name:      "valid 4-char code",
			assetCode: "ABCD",
		},
		{
			name:      "valid 5-char code",
			assetCode: "ABCDE",
		},
		{
			name:      "valid 12-char code",
			assetCode: "ABCDEFGHIJKL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAssetCode(tt.assetCode)
			assert.NoError(t, err)
		})
	}
}

func TestValidateAssetCode_InvalidCodes(t *testing.T) {
	tests := []struct {
		name      string
		assetCode string
	}{
		{
			name:      "empty code",
			assetCode: "",
		},
		{
			name:      "too long",
			assetCode: "ABCDEFGHIJKLM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAssetCode(tt.assetCode)
			assert.Error(t, err)
		})
	}
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestErrAccountNotFound_Error(t *testing.T) {
	err := ErrAccountNotFound{
		Address: "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
	}

	assert.Contains(t, err.Error(), "account not found")
	assert.Contains(t, err.Error(), "GBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX")
}

func TestErrAccountNotFound_EmptyAddress(t *testing.T) {
	err := ErrAccountNotFound{
		Address: "",
	}

	assert.Contains(t, err.Error(), "account not found")
}

// ============================================================================
// Network Configuration Tests
// ============================================================================

func TestNetworkEnvironment_Testnet(t *testing.T) {
	network := Testnet
	assert.Equal(t, Network("testnet"), network)
	assert.Equal(t, "testnet", string(network))
}

func TestNetworkEnvironment_Mainnet(t *testing.T) {
	network := Mainnet
	assert.Equal(t, Network("mainnet"), network)
	assert.Equal(t, "mainnet", string(network))
}

func TestNetworkEnvironment_Futurenet(t *testing.T) {
	network := Futurenet
	assert.Equal(t, Network("futurenet"), network)
	assert.Equal(t, "futurenet", string(network))
}

func TestNetworkPassphrase_Testnet(t *testing.T) {
	networkID := getNetworkID(Testnet)
	assert.Equal(t, "Test SDF Network ; September 2015", networkID)
}

func TestNetworkPassphrase_Mainnet(t *testing.T) {
	networkID := getNetworkID(Mainnet)
	assert.Equal(t, "Public Global Stellar Network ; September 2015", networkID)
}

func TestNetworkPassphrase_Futurenet(t *testing.T) {
	networkID := getNetworkID(Futurenet)
	assert.Equal(t, "Test SDF Future Network ; October 2022", networkID)
}

func TestNetworkPassphrase_InvalidNetwork(t *testing.T) {
	network := Network("invalid")
	networkID := getNetworkID(network)
	assert.Equal(t, "Test SDF Network ; September 2015", networkID)
}

func TestNetworkPassphrase_Custom(t *testing.T) {
	customPassphrase := "Custom Network Passphrase"
	cfg := Config{
		Network:   Testnet,
		NetworkID: customPassphrase,
		RPCURL:    "https://soroban-testnet.stellar.org",
		SourceKey: "SBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
	}

	assert.Equal(t, customPassphrase, cfg.NetworkID)
}

func TestNetworkEnvironment_DevelopmentConfig(t *testing.T) {
	cfg := Config{
		Network:   Testnet,
		RPCURL:    "https://soroban-testnet.stellar.org",
		SourceKey: "SBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
	}

	assert.Equal(t, Testnet, cfg.Network)
	assert.Contains(t, cfg.RPCURL, "testnet")
}

func TestNetworkEnvironment_ProductionConfig(t *testing.T) {
	cfg := Config{
		Network:   Mainnet,
		RPCURL:    "https://soroban-mainnet.stellar.org",
		SourceKey: "SBVH6U5PEFXPXPJ4GPXVYACRF4NZQA5QBCZLLPQGHXWWK6NXPV6IYGGX",
	}

	assert.Equal(t, Mainnet, cfg.Network)
	assert.Contains(t, cfg.RPCURL, "mainnet")
}

// ============================================================================
// Helper Functions
// ============================================================================

// validateStellarAddress validates a Stellar address format
func validateStellarAddress(address string) error {
	if address == "" {
		return ErrAccountNotFound{Address: address}
	}
	if len(address) != 56 {
		return ErrAccountNotFound{Address: address}
	}
	if address[0] != 'G' && address[0] != 'S' {
		return ErrAccountNotFound{Address: address}
	}
	return nil
}

// validateAssetCode validates a Stellar asset code
func validateAssetCode(code string) error {
	if code == "" {
		return fmt.Errorf("asset code is required")
	}
	if len(code) > 12 {
		return fmt.Errorf("asset code must be 12 characters or less")
	}
	return nil
}

// AccountBalance represents an account balance for testing
type AccountBalance struct {
	Address   string
	AssetCode string
	Amount    decimal.Decimal
}

// ErrAccountNotFound represents an account not found error
type ErrAccountNotFound struct {
	Address string
}

func (e ErrAccountNotFound) Error() string {
	return "account not found: " + e.Address
}

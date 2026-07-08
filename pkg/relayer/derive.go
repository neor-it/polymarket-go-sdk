package relayer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// The deposit wallet factory deploys per-owner deposit wallets as one of two
// ERC-1967 clone shapes:
//
//   - UUPS clones (legacy): each wallet stores its own implementation address
//     in its ERC-1967 implementation slot. Used by wallets deployed before the
//     factory's beacon upgrade; those wallets remain functional at their
//     original address.
//   - BeaconProxy clones (current): wallets resolve their implementation by
//     reading a shared beacon contract instead of their own storage. Used by
//     the factory for all new deployments since the upgrade.
//
// The two shapes hash to different CREATE2 addresses for the same owner, so a
// caller that only computes one must check both before concluding a wallet
// doesn't exist — see ResolveDepositWallet. This mirrors Polymarket's own
// resolution algorithm:
// https://docs.polymarket.com/api-reference/relayer/deposit-wallets#wallet-implementation
const (
	// ImplementationPolygon is the deposit wallet implementation contract the
	// factory baked into every ERC-1967 UUPS clone on Polygon mainnet (137)
	// before the factory's beacon upgrade. Only relevant for deriving wallets
	// that predate the upgrade — new wallets use BeaconPolygon instead.
	// Verified against the live deploy
	// 0xf2d22487bd9831cde03764f3bf13604df438be432dca8cfac95b6ff3d66fc7c4
	// (implementation read from the wallet's ERC-1967 implementation slot).
	ImplementationPolygon = "0x58Ca52EbE0DadfdF531cdE7062E76746de4Db1eB"

	// BeaconPolygon is the shared beacon that current (post-upgrade) deposit
	// wallets on Polygon mainnet (137) delegate to. Verified live via the
	// factory's BEACON() view (selector 0x49493a4d), which returns this exact
	// address.
	BeaconPolygon = "0x7A18EDfe055488A3128f01F563e5B479D92ffc3a"
)

// Byte constants from Solady v0.1.26 LibClone.initCodeHashERC1967 /
// initCodeHashERC1967BeaconProxy, mirrored from Polymarket's
// builder-relayer-client derive.ts (package @polymarket/builder-relayer-client,
// dist/builder/derive.js). deriveUupsDepositWallet there is deprecated in favor
// of a shape-aware resolver — ResolveDepositWallet is this SDK's equivalent.
var (
	erc1967Const1 = common.Hex2Bytes("cc3735a920a3ca505d382bbc545af43d6000803e6038573d6000fd5b3d6000f3")
	erc1967Const2 = common.Hex2Bytes("5155f3363d3d373d3d363d7f360894a13ba1a3210667c828492db98dca3e2076")
	erc1967Prefix = new(big.Int).SetBytes(common.Hex2Bytes("61003d3d8160233d3973"))

	erc1967BeaconConst1 = common.Hex2Bytes("b3582b35133d50545afa5036515af43d6000803e604d573d6000fd5b3d6000f3")
	erc1967BeaconConst2 = common.Hex2Bytes("1b60e01b36527fa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6c")
	erc1967BeaconConst3 = common.Hex2Bytes("60195155f3363d3d373d3d363d602036600436635c60da")
	erc1967BeaconPrefix = new(big.Int).SetBytes(common.Hex2Bytes("6100523d8160233d3973"))
)

// depositWalletArgs builds abi.encode(address factory, bytes32 walletId), the
// CREATE2 salt preimage and immutable-args suffix shared by both clone shapes.
func depositWalletArgs(owner, factory common.Address) []byte {
	args := make([]byte, 0, 64)
	args = append(args, common.LeftPadBytes(factory.Bytes(), 32)...)
	args = append(args, common.LeftPadBytes(owner.Bytes(), 32)...)
	return args
}

// DeriveUupsDepositWallet computes the deterministic CREATE2 address of the
// legacy ERC-1967 UUPS deposit wallet clone for owner:
//
//	walletId     = bytes32(owner)
//	args         = abi.encode(factory, walletId)
//	salt         = keccak256(args)
//	bytecodeHash = Solady LibClone.initCodeHashERC1967(implementation, args)
//	wallet       = CREATE2(factory, salt, bytecodeHash)
//
// Only wallets deployed before the factory's beacon upgrade live here — new
// wallets use DeriveBeaconDepositWallet. Callers that don't know a wallet's
// vintage should use ResolveDepositWallet instead of calling this directly.
func DeriveUupsDepositWallet(owner, factory, implementation common.Address) common.Address {
	args := depositWalletArgs(owner, factory)
	salt := ethcrypto.Keccak256(args)
	bytecodeHash := initCodeHashERC1967(implementation, args)
	return ethcrypto.CreateAddress2(factory, common.BytesToHash(salt), bytecodeHash)
}

// DeriveBeaconDepositWallet computes the deterministic CREATE2 address of the
// current ERC-1967 BeaconProxy deposit wallet clone for owner. Same CREATE2
// salt as DeriveUupsDepositWallet; only the bytecode hash (and therefore the
// resulting address) differs, since it resolves its implementation from the
// beacon instead of baking one into its own storage.
func DeriveBeaconDepositWallet(owner, factory, beacon common.Address) common.Address {
	args := depositWalletArgs(owner, factory)
	salt := ethcrypto.Keccak256(args)
	bytecodeHash := initCodeHashERC1967Beacon(beacon, args)
	return ethcrypto.CreateAddress2(factory, common.BytesToHash(salt), bytecodeHash)
}

// ResolveDepositWallet finds owner's actual deployed deposit wallet address by
// checking both clone shapes against the relayer's /deployed registry (via
// client.IsWalletDeployed), current shape first. Use this instead of a single
// Derive* call whenever a wallet's clone shape is unknown — e.g. after
// submitting WALLET-CREATE and polling for confirmation, or when adopting a
// wallet deployed by an earlier run.
//
// Returns the beacon-shape address with deployed=false if neither shape has
// been deployed on-chain yet.
func ResolveDepositWallet(ctx context.Context, client Client, owner common.Address) (wallet common.Address, deployed bool, err error) {
	factory := common.HexToAddress(FactoryPolygon)
	beacon := common.HexToAddress(BeaconPolygon)
	implementation := common.HexToAddress(ImplementationPolygon)

	beaconWallet := DeriveBeaconDepositWallet(owner, factory, beacon)
	if ok, checkErr := client.IsWalletDeployed(ctx, beaconWallet); checkErr == nil && ok {
		return beaconWallet, true, nil
	}

	uupsWallet := DeriveUupsDepositWallet(owner, factory, implementation)
	if ok, checkErr := client.IsWalletDeployed(ctx, uupsWallet); checkErr == nil && ok {
		return uupsWallet, true, nil
	}

	return beaconWallet, false, nil
}

// initCodeHashERC1967 replicates Solady LibClone.initCodeHashERC1967:
// keccak256(prefix+(n<<56) (10 bytes) ‖ implementation(20) ‖ 0x6009 ‖ const2(32) ‖ const1(32) ‖ args).
func initCodeHashERC1967(implementation common.Address, args []byte) []byte {
	combined := new(big.Int).Add(
		erc1967Prefix,
		new(big.Int).Lsh(big.NewInt(int64(len(args))), 56),
	)
	prefix := make([]byte, 10)
	combined.FillBytes(prefix)

	preimage := make([]byte, 0, 10+20+2+32+32+len(args))
	preimage = append(preimage, prefix...)
	preimage = append(preimage, implementation.Bytes()...)
	preimage = append(preimage, 0x60, 0x09)
	preimage = append(preimage, erc1967Const2...)
	preimage = append(preimage, erc1967Const1...)
	preimage = append(preimage, args...)

	return ethcrypto.Keccak256(preimage)
}

// initCodeHashERC1967Beacon replicates Solady LibClone.initCodeHashERC1967BeaconProxy:
// keccak256(prefix+(n<<56) (10 bytes) ‖ beacon(20) ‖ const3 ‖ const2(32) ‖ const1(32) ‖ args).
func initCodeHashERC1967Beacon(beacon common.Address, args []byte) []byte {
	combined := new(big.Int).Add(
		erc1967BeaconPrefix,
		new(big.Int).Lsh(big.NewInt(int64(len(args))), 56),
	)
	prefix := make([]byte, 10)
	combined.FillBytes(prefix)

	preimage := make([]byte, 0, 10+20+len(erc1967BeaconConst3)+32+32+len(args))
	preimage = append(preimage, prefix...)
	preimage = append(preimage, beacon.Bytes()...)
	preimage = append(preimage, erc1967BeaconConst3...)
	preimage = append(preimage, erc1967BeaconConst2...)
	preimage = append(preimage, erc1967BeaconConst1...)
	preimage = append(preimage, args...)

	return ethcrypto.Keccak256(preimage)
}

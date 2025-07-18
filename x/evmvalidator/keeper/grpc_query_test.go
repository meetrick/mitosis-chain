package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/mitosis-org/chain/x/evmvalidator/keeper"
	"github.com/mitosis-org/chain/x/evmvalidator/testutil"
	"github.com/mitosis-org/chain/x/evmvalidator/types"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// QueryTestSuite is a test suite for GRPC queries
type QueryTestSuite struct {
	suite.Suite
	tk          testutil.TestKeeper
	queryServer types.QueryServer
}

// SetupTest initializes the test suite
func (s *QueryTestSuite) SetupTest() {
	s.tk = testutil.NewTestKeeper(&s.Suite)
	s.queryServer = keeper.NewQueryServer(s.tk.Keeper)

	// Set up params
	s.tk.SetupDefaultTestParams()
}

// SetupTestCollateralOwnership sets up a test environment for collateral ownership testing
// by removing any existing ownership for the validator
func (s *QueryTestSuite) SetupTestCollateralOwnership(validator types.Validator) types.Validator {
	// Remove default ownership created by RegisterTestValidator
	s.tk.Keeper.DeleteCollateralOwnership(s.tk.Ctx, validator.Addr, validator.Addr)

	// Reset collateral shares to 0
	validator.CollateralShares = math.ZeroUint()
	s.tk.Keeper.SetValidator(s.tk.Ctx, validator)

	return validator
}

// TestQueryTestSuite runs the query test suite
func TestQueryTestSuite(t *testing.T) {
	suite.Run(t, new(QueryTestSuite))
}

// Test_QueryParams tests the Params query
func (s *QueryTestSuite) Test_QueryParams() {
	resp, err := s.queryServer.Params(s.tk.Ctx, &types.QueryParamsRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	params := s.tk.Keeper.GetParams(s.tk.Ctx)
	s.Require().Equal(params, resp.Params)

	// Test nil request
	_, err = s.queryServer.Params(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// Test_QueryValidatorEntrypointContractAddr tests the ValidatorEntrypointContractAddr query
func (s *QueryTestSuite) Test_QueryValidatorEntrypointContractAddr() {
	// Set the validator entrypoint contract address
	_, _, ethAddr := testutil.GenerateSecp256k1Key()
	s.tk.Keeper.SetValidatorEntrypointContractAddr(s.tk.Ctx, ethAddr)

	resp, err := s.queryServer.ValidatorEntrypointContractAddr(s.tk.Ctx, &types.QueryValidatorEntrypointContractAddrRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(ethAddr, resp.Addr)

	// Test nil request
	_, err = s.queryServer.ValidatorEntrypointContractAddr(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// Test_QueryValidator tests the Validator query
func (s *QueryTestSuite) Test_QueryValidator() {
	// Register a validator
	validator := s.tk.RegisterTestValidator(math.NewUint(1000000000), math.ZeroUint(), false)

	// Query the validator
	req := &types.QueryValidatorRequest{
		ValAddr: validator.Addr.Bytes(),
	}
	resp, err := s.queryServer.Validator(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(validator, resp.Validator)

	// Test nil request
	_, err = s.queryServer.Validator(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test nil validator address
	_, err = s.queryServer.Validator(s.tk.Ctx, &types.QueryValidatorRequest{})
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test non-existent validator
	_, _, nonExistentAddr := testutil.GenerateSecp256k1Key()
	_, err = s.queryServer.Validator(s.tk.Ctx, &types.QueryValidatorRequest{
		ValAddr: nonExistentAddr.Bytes(),
	})
	s.Require().Error(err)
	s.Require().Equal(codes.NotFound, status.Code(err))
}

// Test_QueryValidators tests the Validators query
func (s *QueryTestSuite) Test_QueryValidators() {
	// Register multiple validators
	val1 := s.tk.RegisterTestValidator(math.NewUint(1000000000), math.ZeroUint(), false)
	val2 := s.tk.RegisterTestValidator(math.NewUint(2000000000), math.ZeroUint(), false)
	val3 := s.tk.RegisterTestValidator(math.NewUint(3000000000), math.ZeroUint(), false)

	expectedValidators := []types.Validator{val1, val2, val3}

	// Query all validators
	req := &types.QueryValidatorsRequest{
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err := s.queryServer.Validators(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.Validators, 3)

	// Verify each validator is in the response (not checking order)
	for _, expectedVal := range expectedValidators {
		found := false
		for _, respVal := range resp.Validators {
			if respVal.Addr.String() == expectedVal.Addr.String() {
				found = true
				s.Require().Equal(expectedVal, respVal)
				break
			}
		}
		s.Require().True(found, "Validator %s not found in response", expectedVal.Addr.String())
	}

	// Test nil request
	_, err = s.queryServer.Validators(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test pagination (limit to 2)
	reqWithPagination := &types.QueryValidatorsRequest{
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	}
	respWithPagination, err := s.queryServer.Validators(s.tk.Ctx, reqWithPagination)
	s.Require().NoError(err)
	s.Require().NotNil(respWithPagination)
	s.Require().Len(respWithPagination.Validators, 2)
	s.Require().NotNil(respWithPagination.Pagination.NextKey, "Should have pagination next key")
}

// Test_QueryWithdrawal tests the Withdrawal query
func (s *QueryTestSuite) Test_QueryWithdrawal() {
	// Register a validator
	validator := s.tk.RegisterTestValidator(math.NewUint(10000000000), math.ZeroUint(), false) // 10 MITO

	// Create a withdrawal
	withdrawal := &types.Withdrawal{
		ValAddr:        validator.Addr,
		Amount:         1000000000, // Withdraw 1 MITO
		Receiver:       validator.Addr,
		MaturesAt:      s.tk.Ctx.BlockTime().Unix() + 86400, // 1 day from now
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}

	// Add withdrawal with ID
	s.tk.Keeper.AddNewWithdrawalWithNextID(s.tk.Ctx, withdrawal)

	// Query the withdrawal
	req := &types.QueryWithdrawalRequest{
		Id: withdrawal.ID,
	}
	resp, err := s.queryServer.Withdrawal(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(*withdrawal, resp.Withdrawal)

	// Test nil request
	_, err = s.queryServer.Withdrawal(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test zero ID
	_, err = s.queryServer.Withdrawal(s.tk.Ctx, &types.QueryWithdrawalRequest{})
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test non-existent withdrawal
	_, err = s.queryServer.Withdrawal(s.tk.Ctx, &types.QueryWithdrawalRequest{
		Id: 9999, // Non-existent ID
	})
	s.Require().Error(err)
	s.Require().Equal(codes.NotFound, status.Code(err))
}

// Test_QueryWithdrawals tests the Withdrawals query
func (s *QueryTestSuite) Test_QueryWithdrawals() {
	// Register a validator
	validator := s.tk.RegisterTestValidator(math.NewUint(10000000000), math.ZeroUint(), false) // 10 MITO

	// Create withdrawals directly with the keeper's method
	withdrawal1 := &types.Withdrawal{
		ValAddr:        validator.Addr,
		Amount:         1000000000, // Withdraw 1 MITO
		Receiver:       validator.Addr,
		MaturesAt:      s.tk.Ctx.BlockTime().Unix() + 86400, // 1 day from now
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}
	s.tk.Keeper.AddNewWithdrawalWithNextID(s.tk.Ctx, withdrawal1)

	withdrawal2 := &types.Withdrawal{
		ValAddr:        validator.Addr,
		Amount:         2000000000, // Withdraw 2 MITO
		Receiver:       validator.Addr,
		MaturesAt:      s.tk.Ctx.BlockTime().Unix() + 86400*2, // 2 days from now
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}
	s.tk.Keeper.AddNewWithdrawalWithNextID(s.tk.Ctx, withdrawal2)

	// Query all withdrawals
	req := &types.QueryWithdrawalsRequest{
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err := s.queryServer.Withdrawals(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().GreaterOrEqual(len(resp.Withdrawals), 2)

	// Test nil request
	_, err = s.queryServer.Withdrawals(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// Test_QueryWithdrawalsByValidator tests the WithdrawalsByValidator query
func (s *QueryTestSuite) Test_QueryWithdrawalsByValidator() {
	// Register a validator
	validator := s.tk.RegisterTestValidator(math.NewUint(10000000000), math.ZeroUint(), false) // 10 MITO

	// Create withdrawals
	withdrawal1 := &types.Withdrawal{
		ValAddr:        validator.Addr,
		Amount:         1000000000, // Withdraw 1 MITO
		Receiver:       validator.Addr,
		MaturesAt:      s.tk.Ctx.BlockTime().Unix() + 86400, // 1 day from now
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}
	s.tk.Keeper.AddNewWithdrawalWithNextID(s.tk.Ctx, withdrawal1)

	withdrawal2 := &types.Withdrawal{
		ValAddr:        validator.Addr,
		Amount:         2000000000, // Withdraw 2 MITO
		Receiver:       validator.Addr,
		MaturesAt:      s.tk.Ctx.BlockTime().Unix() + 86400*2, // 2 days from now
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}
	s.tk.Keeper.AddNewWithdrawalWithNextID(s.tk.Ctx, withdrawal2)

	// Query withdrawals by validator
	req := &types.QueryWithdrawalsByValidatorRequest{
		ValAddr: validator.Addr.Bytes(),
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err := s.queryServer.WithdrawalsByValidator(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().GreaterOrEqual(len(resp.Withdrawals), 2)

	// Test nil request
	_, err = s.queryServer.WithdrawalsByValidator(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test nil validator address
	_, err = s.queryServer.WithdrawalsByValidator(s.tk.Ctx, &types.QueryWithdrawalsByValidatorRequest{})
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test withdrawals for non-existent validator (should return empty list, not error)
	_, _, nonExistentAddr := testutil.GenerateSecp256k1Key()
	respEmpty, err := s.queryServer.WithdrawalsByValidator(s.tk.Ctx, &types.QueryWithdrawalsByValidatorRequest{
		ValAddr: nonExistentAddr.Bytes(),
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	s.Require().NoError(err)
	s.Require().NotNil(respEmpty)
	s.Require().Len(respEmpty.Withdrawals, 0)
}

// Test_QueryCollateralOwnershipsByValidator tests the CollateralOwnershipsByValidator query
func (s *QueryTestSuite) Test_QueryCollateralOwnershipsByValidator() {
	// Register a validator without default ownership
	validator := s.tk.RegisterTestValidator(math.NewUint(1000000000), math.ZeroUint(), false)

	// Set up test environment for collateral ownership
	validator = s.SetupTestCollateralOwnership(validator)

	// Set validator's collateral shares for the test
	totalShares := math.NewUint(1000000000) // Same as collateral for 1:1 ratio
	validator.CollateralShares = totalShares
	s.tk.Keeper.SetValidator(s.tk.Ctx, validator)

	// Create owners
	_, _, owner1 := testutil.GenerateSecp256k1Key()
	_, _, owner2 := testutil.GenerateSecp256k1Key()

	// Calculate shares based on precision
	owner1Shares := totalShares.MulUint64(40).QuoUint64(100) // 40% of total shares
	owner2Shares := totalShares.MulUint64(60).QuoUint64(100) // 60% of total shares

	// Create ownership records
	ownership1 := types.CollateralOwnership{
		ValAddr:        validator.Addr,
		Owner:          owner1,
		Shares:         owner1Shares,
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}

	ownership2 := types.CollateralOwnership{
		ValAddr:        validator.Addr,
		Owner:          owner2,
		Shares:         owner2Shares,
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}

	// Set the ownership records
	s.tk.Keeper.SetCollateralOwnership(s.tk.Ctx, ownership1)
	s.tk.Keeper.SetCollateralOwnership(s.tk.Ctx, ownership2)

	// Query collateral ownerships by validator
	req := &types.QueryCollateralOwnershipsByValidatorRequest{
		ValAddr: validator.Addr.Bytes(),
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err := s.queryServer.CollateralOwnershipsByValidator(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.CollateralOwnerships, 2)

	// Expected collateral amounts
	owner1ExpectedAmount := validator.Collateral.MulUint64(40).QuoUint64(100) // 40% of total collateral
	owner2ExpectedAmount := validator.Collateral.MulUint64(60).QuoUint64(100) // 60% of total collateral

	// Verify the amounts
	for _, ownership := range resp.CollateralOwnerships {
		if ownership.Ownership.Owner.String() == owner1.String() {
			s.Require().Equal(owner1ExpectedAmount, ownership.Amount, "Owner1 should have 40% of the collateral")
		} else if ownership.Ownership.Owner.String() == owner2.String() {
			s.Require().Equal(owner2ExpectedAmount, ownership.Amount, "Owner2 should have 60% of the collateral")
		}
	}

	// Test nil request
	_, err = s.queryServer.CollateralOwnershipsByValidator(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test nil validator address
	_, err = s.queryServer.CollateralOwnershipsByValidator(s.tk.Ctx, &types.QueryCollateralOwnershipsByValidatorRequest{})
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test non-existent validator
	_, _, nonExistentAddr := testutil.GenerateSecp256k1Key()
	_, err = s.queryServer.CollateralOwnershipsByValidator(s.tk.Ctx, &types.QueryCollateralOwnershipsByValidatorRequest{
		ValAddr: nonExistentAddr.Bytes(),
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	s.Require().Error(err)
	s.Require().Equal(codes.NotFound, status.Code(err))
}

// Test_QueryCollateralOwnerships tests the CollateralOwnerships query
func (s *QueryTestSuite) Test_QueryCollateralOwnerships() {
	// Register multiple validators without default ownership
	val1 := s.tk.RegisterTestValidator(math.NewUint(1000000000), math.ZeroUint(), false)
	val2 := s.tk.RegisterTestValidator(math.NewUint(2000000000), math.ZeroUint(), false)

	// Set up test environment for collateral ownership
	val1 = s.SetupTestCollateralOwnership(val1)
	val2 = s.SetupTestCollateralOwnership(val2)

	// Set validators' collateral shares for the test
	val1TotalShares := math.NewUint(1000000000) // Same as collateral for 1:1 ratio
	val1.CollateralShares = val1TotalShares
	s.tk.Keeper.SetValidator(s.tk.Ctx, val1)

	val2TotalShares := math.NewUint(2000000000) // Same as collateral for 1:1 ratio
	val2.CollateralShares = val2TotalShares
	s.tk.Keeper.SetValidator(s.tk.Ctx, val2)

	// Create owners
	_, _, owner1 := testutil.GenerateSecp256k1Key()
	_, _, owner2 := testutil.GenerateSecp256k1Key()
	_, _, owner3 := testutil.GenerateSecp256k1Key()

	// Calculate shares based on precision
	owner1Shares := val1TotalShares.MulUint64(50).QuoUint64(100) // 50% of val1's shares
	owner2Shares := val1TotalShares.MulUint64(50).QuoUint64(100) // 50% of val1's shares
	owner3Shares := val2TotalShares                              // 100% of val2's shares

	// Create ownership records
	ownership1 := types.CollateralOwnership{
		ValAddr:        val1.Addr,
		Owner:          owner1,
		Shares:         owner1Shares,
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}

	ownership2 := types.CollateralOwnership{
		ValAddr:        val1.Addr,
		Owner:          owner2,
		Shares:         owner2Shares,
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}

	ownership3 := types.CollateralOwnership{
		ValAddr:        val2.Addr,
		Owner:          owner3,
		Shares:         owner3Shares,
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}

	// Set the ownership records
	s.tk.Keeper.SetCollateralOwnership(s.tk.Ctx, ownership1)
	s.tk.Keeper.SetCollateralOwnership(s.tk.Ctx, ownership2)
	s.tk.Keeper.SetCollateralOwnership(s.tk.Ctx, ownership3)

	// Query all collateral ownerships
	req := &types.QueryCollateralOwnershipsRequest{
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err := s.queryServer.CollateralOwnerships(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Len(resp.CollateralOwnerships, 3)

	// Test nil request
	_, err = s.queryServer.CollateralOwnerships(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test pagination (limit to 2)
	reqWithPagination := &types.QueryCollateralOwnershipsRequest{
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	}
	respWithPagination, err := s.queryServer.CollateralOwnerships(s.tk.Ctx, reqWithPagination)
	s.Require().NoError(err)
	s.Require().NotNil(respWithPagination)
	s.Require().Len(respWithPagination.CollateralOwnerships, 2)
	s.Require().NotNil(respWithPagination.Pagination.NextKey, "Should have pagination next key")
}

// Test_QueryCollateralOwnership tests the CollateralOwnership query
func (s *QueryTestSuite) Test_QueryCollateralOwnership() {
	// Register a validator without default ownership
	validator := s.tk.RegisterTestValidator(math.NewUint(1000000000), math.ZeroUint(), false)

	// Set up test environment for collateral ownership
	validator = s.SetupTestCollateralOwnership(validator)

	// Set validator's collateral shares for the test
	totalShares := math.NewUint(1000000000) // Same as collateral for 1:1 ratio
	validator.CollateralShares = totalShares
	s.tk.Keeper.SetValidator(s.tk.Ctx, validator)

	// Create a custom owner
	_, _, ownerAddr := testutil.GenerateSecp256k1Key()

	// Calculate shares based on precision - 50% of total shares
	ownerShares := totalShares.MulUint64(50).QuoUint64(100)
	expectedAmount := validator.Collateral.MulUint64(50).QuoUint64(100) // 50% of total collateral

	// Add collateral for the custom owner
	ownership := types.CollateralOwnership{
		ValAddr:        validator.Addr,
		Owner:          ownerAddr,
		Shares:         ownerShares,
		CreationHeight: s.tk.Ctx.BlockHeight(),
	}
	s.tk.Keeper.SetCollateralOwnership(s.tk.Ctx, ownership)

	// Query the collateral ownership
	req := &types.QueryCollateralOwnershipRequest{
		ValAddr: validator.Addr.Bytes(),
		Owner:   ownerAddr.Bytes(),
	}
	resp, err := s.queryServer.CollateralOwnership(s.tk.Ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(ownership, resp.CollateralOwnership.Ownership)
	s.Require().Equal(expectedAmount, resp.CollateralOwnership.Amount) // Should be half of the total collateral

	// Test nil request
	_, err = s.queryServer.CollateralOwnership(s.tk.Ctx, nil)
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test nil validator address
	_, err = s.queryServer.CollateralOwnership(s.tk.Ctx, &types.QueryCollateralOwnershipRequest{Owner: ownerAddr.Bytes()})
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test nil owner address
	_, err = s.queryServer.CollateralOwnership(s.tk.Ctx, &types.QueryCollateralOwnershipRequest{ValAddr: validator.Addr.Bytes()})
	s.Require().Error(err)
	s.Require().Equal(codes.InvalidArgument, status.Code(err))

	// Test non-existent validator
	_, _, nonExistentAddr := testutil.GenerateSecp256k1Key()
	_, err = s.queryServer.CollateralOwnership(s.tk.Ctx, &types.QueryCollateralOwnershipRequest{
		ValAddr: nonExistentAddr.Bytes(),
		Owner:   ownerAddr.Bytes(),
	})
	s.Require().Error(err)
	s.Require().Equal(codes.NotFound, status.Code(err))

	// Test non-existent ownership
	_, _, nonExistentOwner := testutil.GenerateSecp256k1Key()
	_, err = s.queryServer.CollateralOwnership(s.tk.Ctx, &types.QueryCollateralOwnershipRequest{
		ValAddr: validator.Addr.Bytes(),
		Owner:   nonExistentOwner.Bytes(),
	})
	s.Require().Error(err)
	s.Require().Equal(codes.NotFound, status.Code(err))
}

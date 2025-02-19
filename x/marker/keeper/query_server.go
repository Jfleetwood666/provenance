package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/store/prefix"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/provenance-io/provenance/x/marker/types"
)

var _ types.QueryServer = Keeper{}

// Params queries params of distribution module
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// AllMarkers returns a list of all markers on the blockchain
func (k Keeper) AllMarkers(c context.Context, req *types.QueryAllMarkersRequest) (*types.QueryAllMarkersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	markers := make([]*codectypes.Any, 0)
	store := ctx.KVStore(k.storeKey)
	markerStore := prefix.NewStore(store, types.MarkerStoreKeyPrefix)
	pageRes, err := query.Paginate(markerStore, req.Pagination, func(_ []byte, value []byte) error {
		result, err := k.GetMarker(ctx, sdk.AccAddress(value))
		if err == nil {
			anyMsg, anyErr := codectypes.NewAnyWithValue(result)
			if anyErr != nil {
				return status.Error(codes.Internal, anyErr.Error())
			}
			markers = append(markers, anyMsg)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryAllMarkersResponse{Markers: markers, Pagination: pageRes}, nil
}

// Marker query for a single marker by denom or address
func (k Keeper) Marker(c context.Context, req *types.QueryMarkerRequest) (*types.QueryMarkerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	marker, err := accountForDenomOrAddress(ctx, k, req.Id)
	if err != nil {
		return nil, err
	}
	anyMsg, err := codectypes.NewAnyWithValue(marker)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryMarkerResponse{Marker: anyMsg}, nil
}

// Holding query for all accounts holding the given marker coins
func (k Keeper) Holding(c context.Context, req *types.QueryHoldingRequest) (*types.QueryHoldingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	marker, err := accountForDenomOrAddress(ctx, k, req.Id)
	if err != nil {
		return nil, err
	}

	denom := marker.GetDenom()
	denomOwners, err := k.bankKeeper.DenomOwners(c, &banktypes.QueryDenomOwnersRequest{
		Denom:      denom,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	balances := make([]types.Balance, len(denomOwners.DenomOwners))
	for i, bal := range denomOwners.DenomOwners {
		balances[i] = types.Balance{
			Address: bal.Address,
			Coins:   sdk.NewCoins(bal.Balance),
		}
	}

	return &types.QueryHoldingResponse{
		Balances:   balances,
		Pagination: denomOwners.Pagination,
	}, nil
}

// Supply query for supply of coin on a marker account
func (k Keeper) Supply(c context.Context, req *types.QuerySupplyRequest) (*types.QuerySupplyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	marker, err := accountForDenomOrAddress(ctx, k, req.Id)
	if err != nil {
		return nil, err
	}
	return &types.QuerySupplyResponse{Amount: marker.GetSupply()}, nil
}

// Escrow query for coins on a marker account
func (k Keeper) Escrow(c context.Context, req *types.QueryEscrowRequest) (*types.QueryEscrowResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	marker, err := accountForDenomOrAddress(ctx, k, req.Id)
	if err != nil {
		return nil, err
	}
	escrow := k.bankKeeper.GetAllBalances(ctx, marker.GetAddress())

	return &types.QueryEscrowResponse{Escrow: escrow}, nil
}

// Access query for access records on an account
func (k Keeper) Access(c context.Context, req *types.QueryAccessRequest) (*types.QueryAccessResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	marker, err := accountForDenomOrAddress(ctx, k, req.Id)
	if err != nil {
		return nil, err
	}
	return &types.QueryAccessResponse{Accounts: marker.GetAccessList()}, nil
}

// DenomMetadata query for metadata on denom
func (k Keeper) DenomMetadata(c context.Context, req *types.QueryDenomMetadataRequest) (*types.QueryDenomMetadataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := sdk.ValidateDenom(req.Denom); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid denom")
	}

	ctx := sdk.UnwrapSDKContext(c)

	metadata, _ := k.bankKeeper.GetDenomMetaData(ctx, req.Denom)

	return &types.QueryDenomMetadataResponse{Metadata: metadata}, nil
}

// AccountData query for account data associated with a denom
func (k Keeper) AccountData(c context.Context, req *types.QueryAccountDataRequest) (*types.QueryAccountDataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := types.MarkerAddress(req.Denom)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	value, err := k.attrKeeper.GetAccountData(ctx, addr.String())
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "could not get %q account data: %v", req.Denom, err)
	}

	return &types.QueryAccountDataResponse{Value: value}, nil
}

// NetAssetValues query for returning net asset values for a marker
func (k Keeper) NetAssetValues(c context.Context, req *types.QueryNetAssetValuesRequest) (*types.QueryNetAssetValuesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	marker, err := accountForDenomOrAddress(ctx, k, req.Id)
	if err != nil {
		return nil, err
	}

	var navs []types.NetAssetValue
	err = k.IterateNetAssetValues(ctx, marker.GetAddress(), func(nav types.NetAssetValue) (stop bool) {
		navs = append(navs, nav)
		return false
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryNetAssetValuesResponse{NetAssetValues: navs}, nil
}

// accountForDenomOrAddress attempts to first get a marker by account address and then by denom.
func accountForDenomOrAddress(ctx sdk.Context, keeper Keeper, lookup string) (types.MarkerAccountI, error) {
	var addrErr, err error
	var addr sdk.AccAddress
	var account types.MarkerAccountI

	// try to parse the argument as an address, if this fails try as a denom string.
	if addr, addrErr = sdk.AccAddressFromBech32(lookup); addrErr != nil {
		account, err = keeper.GetMarkerByDenom(ctx, lookup)
	} else {
		account, err = keeper.GetMarker(ctx, addr)
	}
	if err != nil {
		return nil, types.ErrMarkerNotFound.Wrap("invalid denom or address")
	}
	return account, nil
}

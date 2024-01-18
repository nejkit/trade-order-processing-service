package utils

import (
	"time"

	"trade-order-processing-service/external/bps"
	"trade-order-processing-service/external/ops"
	"trade-order-processing-service/models"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func MapOrderInfoToProto(model models.OrderModel) *ops.OpsOrderInfo {
	return &ops.OpsOrderInfo{
		Id:             uuid.NewString(),
		OrderId:        model.OrderId,
		AccountId:      model.AccountId,
		AssetId:        model.AssetId,
		CurrencyPair:   model.CurrencyPair,
		Direction:      ops.OpsOrderDirection(model.Direction),
		LimitPrice:     model.LimitPrice,
		AskVolume:      model.AskVolume,
		FilledVolume:   model.FilledVolume,
		Type:           ops.OpsOrderType(model.Type),
		FillPrice:      model.FilledVolume,
		CreationDate:   timestamppb.New(time.UnixMilli(model.CreationDate)),
		UpdatedDate:    timestamppb.New(time.UnixMilli(model.UpdatedDate)),
		ExpirationDate: timestamppb.New(time.UnixMilli(model.ExpirationDate)),
		MatchingDate:   timestamppb.New(time.UnixMilli(model.MatchingDate)),
		TransferId:     model.TransferId,
		State:          ops.OpsOrderState(model.State),
		ExchangeId:     model.ExchangeId,
	}
}

func MapProtoOrderInfoToModel(protoModel *ops.OpsOrderInfo) models.OrderModel {
	return models.OrderModel{
		OrderId:        protoModel.OrderId,
		AccountId:      protoModel.AccountId,
		AssetId:        protoModel.AssetId,
		CurrencyPair:   protoModel.CurrencyPair,
		Direction:      int(protoModel.Direction),
		LimitPrice:     protoModel.LimitPrice,
		AskVolume:      protoModel.AskVolume,
		FilledVolume:   protoModel.FilledVolume,
		Type:           int(protoModel.Type),
		FilledPrice:    protoModel.FillPrice,
		CreationDate:   protoModel.CreationDate.AsTime().UTC().UnixMilli(),
		UpdatedDate:    protoModel.CreationDate.AsTime().UTC().UnixMilli(),
		ExpirationDate: protoModel.ExpirationDate.AsTime().UTC().UnixMilli(),
		MatchingDate:   protoModel.MatchingDate.AsTime().UTC().UnixMilli(),
		TransferId:     protoModel.TransferId,
		State:          int(protoModel.State),
		ParentId:       protoModel.ParentId,
		ExchangeId:     protoModel.ExchangeId,
	}
}

func MapBpsErrorToOpsError(err *bps.BpsError) *ops.OpsError {
	if err == nil {
		return nil
	}

	switch err.ErrorCode {
	case bps.BpsErrorCode_BPS_ERROR_CODE_ASSET_NOT_RELATED_TO_ACCOUNT:
		return &ops.OpsError{Message: err.Message, ErrorCode: ops.OpsErrorCode_OPS_ERROR_CODE_ASSET_NOT_RELATED_TO_ACCOUNT}
	case bps.BpsErrorCode_BPS_ERROR_CODE_NOT_EXISTS_ASSET:
		return &ops.OpsError{Message: err.Message, ErrorCode: ops.OpsErrorCode_OPS_ERROR_CODE_ASSSET_NOT_EXISTS}
	case bps.BpsErrorCode_BPS_ERROR_CODE_NOT_ENOUGH_BALANCE:
		return &ops.OpsError{Message: err.Message, ErrorCode: ops.OpsErrorCode_OPS_ERROR_CODE_ASSET_BALANCE_NOT_ENOUGH}
	default:
		return &ops.OpsError{Message: err.Message, ErrorCode: ops.OpsErrorCode_OPS_ERROR_CODE_INTERNAL}
	}

}

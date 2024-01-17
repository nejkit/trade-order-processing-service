package utils

import (
	"time"
	"trade-order-processing-service/external/OPS"
	"trade-order-processing-service/external/balances"
	"trade-order-processing-service/models"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func MapOrderInfoToProto(model models.OrderModel) *OPS.OpsOrderInfo {
	return &OPS.OpsOrderInfo{
		Id:             uuid.NewString(),
		OrderId:        model.OrderId,
		AccountId:      model.AccountId,
		AssetId:        model.AssetId,
		CurrencyPair:   model.CurrencyPair,
		Direction:      OPS.OpsOrderDirection(model.Direction),
		LimitPrice:     model.LimitPrice,
		AskVolume:      model.AskVolume,
		FilledVolume:   model.FilledVolume,
		Type:           OPS.OpsOrderType(model.Type),
		FillPrice:      model.FilledVolume,
		CreationDate:   timestamppb.New(time.UnixMilli(model.CreationDate)),
		UpdatedDate:    timestamppb.New(time.UnixMilli(model.UpdatedDate)),
		ExpirationDate: timestamppb.New(time.UnixMilli(model.ExpirationDate)),
		MatchingDate:   timestamppb.New(time.UnixMilli(model.MatchingDate)),
		TransferId:     model.TransferId,
		State:          OPS.OpsOrderState(model.State),
		ExchangeId:     model.ExchangeId,
	}
}

func MapProtoOrderInfoToModel(protoModel *OPS.OpsOrderInfo) models.OrderModel {
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

func MapBpsErrorToOpsError(err *balances.BpsError) *OPS.OpsError {
	if err == nil {
		return nil
	}

	switch err.ErrorCode {
	case balances.BpsErrorCode_BPS_ERROR_CODE_ASSET_NOT_RELATED_TO_ACCOUNT:
		return &OPS.OpsError{Message: err.Message, ErrorCode: OPS.OpsErrorCode_OPS_ERROR_CODE_ASSET_NOT_RELATED_TO_ACCOUNT}
	case balances.BpsErrorCode_BPS_ERROR_CODE_NOT_EXISTS_ASSET:
		return &OPS.OpsError{Message: err.Message, ErrorCode: OPS.OpsErrorCode_OPS_ERROR_CODE_ASSSET_NOT_EXISTS}
	case balances.BpsErrorCode_BPS_ERROR_CODE_NOT_ENOUGH_BALANCE:
		return &OPS.OpsError{Message: err.Message, ErrorCode: OPS.OpsErrorCode_OPS_ERROR_CODE_ASSET_BALANCE_NOT_ENOUGH}
	default:
		return &OPS.OpsError{Message: err.Message, ErrorCode: OPS.OpsErrorCode_OPS_ERROR_CODE_INTERNAL}
	}

}

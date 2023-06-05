package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wx-shi/utxo-indexer/internal/model"
)

const defaultPageSize = 50

func (s *Server) utxoHandle() func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		var req model.UTXORequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"code": http.StatusBadRequest,
				"msg":  err.Error(),
			})
			return
		}
		if req.PageSize == 0 {
			req.PageSize = defaultPageSize
		}

		reply, err := s.db.GetUTXOByAddress(req.Address, req.Page, req.PageSize)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  err.Error(),
			})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"code": http.StatusOK,
				"data": reply,
			})
		}
	}
}

func (s *Server) heightHandle() func(ctx *gin.Context) {
	return func(ctx *gin.Context) {

		sheight, err := s.db.GetStoreHeight()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  err.Error(),
			})
			return
		}

		nheight, err := s.rpc.GetBlockCount()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"code": http.StatusOK,
			"data": model.HeightReply{
				StoreHeight: sheight,
				NodeHeight:  nheight,
			},
		})
	}
}

func (s *Server) utxoInfoHandle() func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		var req model.UTXOInfoRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"code": http.StatusBadRequest,
				"msg":  err.Error(),
			})
			return
		}

		reply, err := s.db.GetUTXOInfoByKeys(req.Keys)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  err.Error(),
			})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"code": http.StatusOK,
				"data": reply,
			})
		}
	}
}

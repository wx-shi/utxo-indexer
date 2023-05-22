package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wx-shi/utxo-indexer/internal/model"
)

func (s *Server) utxoHandle() func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		var req model.UTXORequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		utxos, amout, err := s.db.GetUTXOByAddress(req.Address)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  err.Error(),
			})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"code": http.StatusOK,
				"data": model.UTXOReply{
					Balance: amout,
					Utxos:   utxos,
				},
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

package handlers

import (
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminApertureSettlement struct {
	ID string `json:"id"`
	ApertureID string `json:"aperture_id"`
	Race string `json:"race"`
	Name string `json:"name"`
	SettlementType string `json:"settlement_type"`
	PopulationCapacity int `json:"population_capacity"`
	Housing int `json:"housing"`
	FoodStorage int `json:"food_storage"`
	Security float64 `json:"security"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func AdminListApertureSettlements(c *gin.Context) {
	if _,ok:=requireAdmin(c);!ok{return}
	rows,err:=db.DB.QueryContext(c.Request.Context(),`SELECT id,aperture_id,race,COALESCE(name,''),settlement_type,population_capacity,housing,food_storage,security,created_at,updated_at FROM aperture_variant_settlements WHERE aperture_id=$1 ORDER BY race`,c.Param("id"))
	if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tải khu định cư tiên khiếu"});return};defer rows.Close();list:=make([]AdminApertureSettlement,0)
	for rows.Next(){var r AdminApertureSettlement;if err:=rows.Scan(&r.ID,&r.ApertureID,&r.Race,&r.Name,&r.SettlementType,&r.PopulationCapacity,&r.Housing,&r.FoodStorage,&r.Security,&r.CreatedAt,&r.UpdatedAt);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc khu định cư"});return};list=append(list,r)}
	if err:=rows.Err();err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc khu định cư"});return};c.JSON(http.StatusOK,list)
}

func validateAdminSettlement(r AdminApertureSettlement) string {
	if strings.TrimSpace(r.Race)==""||len(r.Race)>50||len(r.Name)>100{return "Chủng tộc bắt buộc, tên tối đa 100 ký tự"}
	switch r.SettlementType{case "VILLAGE","TOWN","CITY","FORTRESS":default:return "Loại khu định cư không hợp lệ"}
	if r.PopulationCapacity<0||r.Housing<0||r.FoodStorage<0||math.IsNaN(r.Security)||math.IsInf(r.Security,0)||r.Security<0||r.Security>100{return "Sức chứa, nhà ở, lương thực hoặc an ninh không hợp lệ"}
	return ""
}

func AdminCreateApertureSettlement(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};var req AdminApertureSettlement;if err:=c.ShouldBindJSON(&req);err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Dữ liệu khu định cư không hợp lệ"});return};if message:=validateAdminSettlement(req);message!=""{c.JSON(http.StatusBadRequest,gin.H{"error":message});return}
	var raceExists bool;if err:=db.DB.QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM aperture_variant_humans WHERE aperture_id=$1 AND race=$2)`,c.Param("id"),strings.TrimSpace(req.Race)).Scan(&raceExists);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xác minh chủng tộc"});return};if !raceExists{c.JSON(http.StatusBadRequest,gin.H{"error":"Chủng tộc này chưa có dân cư trong tiên khiếu"});return}
	_,err:=db.DB.ExecContext(c.Request.Context(),`INSERT INTO aperture_variant_settlements(aperture_id,race,name,settlement_type,population_capacity,housing,food_storage,security) VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8)`,c.Param("id"),strings.TrimSpace(req.Race),strings.TrimSpace(req.Name),req.SettlementType,req.PopulationCapacity,req.Housing,req.FoodStorage,req.Security);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tạo khu định cư: "+err.Error()});return};auditlog.Logf(db.DB,&adminID,"admin.create_aperture_settlement","aperture=%s race=%s",c.Param("id"),req.Race);c.JSON(http.StatusOK,gin.H{"message":"Đã tạo khu định cư"})
}

func AdminUpdateApertureSettlement(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};var req AdminApertureSettlement;if err:=c.ShouldBindJSON(&req);err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Dữ liệu khu định cư không hợp lệ"});return};if message:=validateAdminSettlement(req);message!=""{c.JSON(http.StatusBadRequest,gin.H{"error":message});return}
	var raceExists bool;if err:=db.DB.QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM aperture_variant_humans WHERE aperture_id=$1 AND race=$2)`,c.Param("id"),strings.TrimSpace(req.Race)).Scan(&raceExists);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xác minh chủng tộc"});return};if !raceExists{c.JSON(http.StatusBadRequest,gin.H{"error":"Chủng tộc này chưa có dân cư trong tiên khiếu"});return}
	result,err:=db.DB.ExecContext(c.Request.Context(),`UPDATE aperture_variant_settlements SET race=$1,name=NULLIF($2,''),settlement_type=$3,population_capacity=$4,housing=$5,food_storage=$6,security=$7,updated_at=NOW() WHERE id=$8 AND aperture_id=$9`,strings.TrimSpace(req.Race),strings.TrimSpace(req.Name),req.SettlementType,req.PopulationCapacity,req.Housing,req.FoodStorage,req.Security,c.Param("settlement_id"),c.Param("id"));if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể cập nhật khu định cư: "+err.Error()});return};if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy khu định cư"});return};auditlog.Logf(db.DB,&adminID,"admin.update_aperture_settlement","id=%s",c.Param("settlement_id"));c.JSON(http.StatusOK,gin.H{"message":"Đã cập nhật khu định cư"})
}

func AdminDeleteApertureSettlement(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};result,err:=db.DB.ExecContext(c.Request.Context(),`DELETE FROM aperture_variant_settlements WHERE id=$1 AND aperture_id=$2`,c.Param("settlement_id"),c.Param("id"));if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xóa khu định cư"});return};if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy khu định cư"});return};auditlog.Logf(db.DB,&adminID,"admin.delete_aperture_settlement","id=%s",c.Param("settlement_id"));c.JSON(http.StatusOK,gin.H{"message":"Đã xóa khu định cư"})
}

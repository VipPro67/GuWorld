package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminApertureEvent struct {
	ID string `json:"id"`
	ApertureID string `json:"aperture_id"`
	EventType string `json:"event_type"`
	Race string `json:"race"`
	Severity string `json:"severity"`
	Description string `json:"description"`
	Effects json.RawMessage `json:"effects"`
	Resolved bool `json:"resolved"`
	CreatedAt time.Time `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}

func AdminListApertureEvents(c *gin.Context) {
	if _,ok:=requireAdmin(c);!ok{return}
	rows,err:=db.DB.QueryContext(c.Request.Context(),`SELECT id,aperture_id,event_type,COALESCE(race,''),severity,COALESCE(description,''),effects,resolved,created_at,resolved_at FROM aperture_variant_events WHERE aperture_id=$1 ORDER BY resolved,created_at DESC`,c.Param("id"))
	if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tải sự kiện xã hội"});return};defer rows.Close();events:=make([]AdminApertureEvent,0)
	for rows.Next(){var e AdminApertureEvent;if err:=rows.Scan(&e.ID,&e.ApertureID,&e.EventType,&e.Race,&e.Severity,&e.Description,&e.Effects,&e.Resolved,&e.CreatedAt,&e.ResolvedAt);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc sự kiện xã hội"});return};events=append(events,e)}
	if err:=rows.Err();err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc sự kiện xã hội"});return};c.JSON(http.StatusOK,events)
}

func validateAdminApertureEvent(c *gin.Context,e *AdminApertureEvent) bool {
	e.EventType=strings.TrimSpace(e.EventType);e.Race=strings.TrimSpace(e.Race);e.Severity=strings.ToUpper(strings.TrimSpace(e.Severity))
	switch e.EventType{case models.EventTypeRebellion,models.EventTypeGeniusBirth,models.EventTypeImmortalAscend,models.EventTypeMigration,models.EventTypeDisaster,models.EventTypeFoodShortage:default:c.JSON(http.StatusBadRequest,gin.H{"error":"Loại sự kiện không hợp lệ"});return false}
	switch e.Severity{case "LOW","MEDIUM","HIGH","CRITICAL":default:c.JSON(http.StatusBadRequest,gin.H{"error":"Mức độ sự kiện không hợp lệ"});return false}
	if len(e.Race)>50||len(e.Description)>5000{c.JSON(http.StatusBadRequest,gin.H{"error":"Chủng tộc hoặc mô tả vượt giới hạn độ dài"});return false}
	if len(e.Effects)==0{e.Effects=json.RawMessage(`{"schema_version":2,"blocks":[]}`)}
	var effects map[string]json.RawMessage;if json.Unmarshal(e.Effects,&effects)!=nil||effects==nil{c.JSON(http.StatusBadRequest,gin.H{"error":"effects phải là JSON object hợp lệ"});return false}
	if e.Race!=""{var exists bool;if err:=db.DB.QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM aperture_variant_humans WHERE aperture_id=$1 AND race=$2)`,c.Param("id"),e.Race).Scan(&exists);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xác minh chủng tộc của sự kiện"});return false};if !exists{c.JSON(http.StatusBadRequest,gin.H{"error":"Chủng tộc không tồn tại trong tiên khiếu này"});return false}}
	return true
}

func AdminCreateApertureEvent(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};var e AdminApertureEvent;if err:=c.ShouldBindJSON(&e);err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Dữ liệu sự kiện không hợp lệ"});return};if !validateAdminApertureEvent(c,&e){return}
	var resolvedAt interface{};if e.Resolved{if e.ResolvedAt!=nil{resolvedAt=*e.ResolvedAt}else{resolvedAt=time.Now()}}
	_,err:=db.DB.ExecContext(c.Request.Context(),`INSERT INTO aperture_variant_events(aperture_id,event_type,race,severity,description,effects,resolved,resolved_at) VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8)`,c.Param("id"),e.EventType,e.Race,e.Severity,e.Description,e.Effects,e.Resolved,resolvedAt);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tạo sự kiện xã hội: "+err.Error()});return};auditlog.Logf(db.DB,&adminID,"admin.create_aperture_event","aperture=%s event=%s",c.Param("id"),e.EventType);c.JSON(http.StatusOK,gin.H{"message":"Đã tạo sự kiện xã hội"})
}

func AdminUpdateApertureEvent(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};var e AdminApertureEvent;if err:=c.ShouldBindJSON(&e);err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Dữ liệu sự kiện không hợp lệ"});return};if !validateAdminApertureEvent(c,&e){return}
	var resolvedAt interface{};if e.Resolved{if e.ResolvedAt!=nil{resolvedAt=*e.ResolvedAt}else{resolvedAt=time.Now()}}
	result,err:=db.DB.ExecContext(c.Request.Context(),`UPDATE aperture_variant_events SET event_type=$1,race=NULLIF($2,''),severity=$3,description=$4,effects=$5,resolved=$6,resolved_at=$7 WHERE id=$8 AND aperture_id=$9`,e.EventType,e.Race,e.Severity,e.Description,e.Effects,e.Resolved,resolvedAt,c.Param("event_id"),c.Param("id"));if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể cập nhật sự kiện xã hội: "+err.Error()});return};if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy sự kiện xã hội"});return};auditlog.Logf(db.DB,&adminID,"admin.update_aperture_event","event=%s",c.Param("event_id"));c.JSON(http.StatusOK,gin.H{"message":"Đã cập nhật sự kiện xã hội"})
}

func AdminDeleteApertureEvent(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};result,err:=db.DB.ExecContext(c.Request.Context(),`DELETE FROM aperture_variant_events WHERE id=$1 AND aperture_id=$2`,c.Param("event_id"),c.Param("id"));if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xóa sự kiện xã hội"});return};if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy sự kiện xã hội"});return};auditlog.Logf(db.DB,&adminID,"admin.delete_aperture_event","event=%s",c.Param("event_id"));c.JSON(http.StatusOK,gin.H{"message":"Đã xóa sự kiện xã hội"})
}

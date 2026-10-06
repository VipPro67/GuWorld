package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListResidualTraces(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok { return }
	page:=1;limit:=50;_,_=fmt.Sscanf(c.DefaultQuery("page","1"),"%d",&page);_,_=fmt.Sscanf(c.DefaultQuery("limit","50"),"%d",&limit);if page<1{page=1};if limit<1||limit>200{limit=50};search:=strings.TrimSpace(c.Query("search"));pattern:="%"+search+"%"
	var total int;if err:=db.DB.QueryRowContext(c.Request.Context(),`SELECT COUNT(*) FROM world_residual_traces t JOIN players p ON p.id=t.player_id WHERE $1='' OR t.region_id ILIKE $2 OR t.action_type ILIKE $2 OR t.dominant_path ILIKE $2 OR p.username ILIKE $2 OR t.player_id::text ILIKE $2`,search,pattern).Scan(&total);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đếm dấu vết thế giới"});return}
	rows,err:=db.DB.QueryContext(c.Request.Context(),`SELECT t.id,t.region_id,t.x,t.y,t.player_id,p.username,t.action_type,t.dominant_path,t.dao_pulse_strength,t.expires_at,t.created_at FROM world_residual_traces t JOIN players p ON p.id=t.player_id WHERE $1='' OR t.region_id ILIKE $2 OR t.action_type ILIKE $2 OR t.dominant_path ILIKE $2 OR p.username ILIKE $2 OR t.player_id::text ILIKE $2 ORDER BY t.created_at DESC,t.id LIMIT $3 OFFSET $4`,search,pattern,limit,(page-1)*limit);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tải dấu vết thế giới"});return};defer rows.Close();data:=make([]gin.H,0)
	for rows.Next(){var id,regionID,playerID,username,actionType,path string;var x,y int;var pulse float64;var expires,created time.Time;if err:=rows.Scan(&id,&regionID,&x,&y,&playerID,&username,&actionType,&path,&pulse,&expires,&created);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc dấu vết thế giới"});return};data=append(data,gin.H{"id":id,"region_id":regionID,"x":x,"y":y,"player_id":playerID,"player_username":username,"action_type":actionType,"dominant_path":path,"dao_pulse_strength":pulse,"expires_at":expires,"created_at":created})};if err:=rows.Err();err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc dấu vết thế giới"});return};c.JSON(http.StatusOK,gin.H{"data":data,"total":total,"page":page,"limit":limit})
}

func AdminUpdateResidualTrace(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};traceID,err:=uuid.Parse(c.Param("id"));if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"ID dấu vết không hợp lệ"});return}
	var req struct{RegionID *string `json:"region_id"`;X *int `json:"x"`;Y *int `json:"y"`;PlayerID *string `json:"player_id"`;ActionType *string `json:"action_type"`;DominantPath *string `json:"dominant_path"`;DaoPulseStrength *float64 `json:"dao_pulse_strength"`;ExpiresAt *string `json:"expires_at"`}
	if err:=c.ShouldBindJSON(&req);err!=nil||req.RegionID==nil||req.X==nil||req.Y==nil||req.PlayerID==nil||req.ActionType==nil||req.DominantPath==nil||req.DaoPulseStrength==nil||req.ExpiresAt==nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Cần đủ dữ liệu vị trí, chủ thể, loại hành động, đạo mạch và hạn dùng"});return}
	region:=strings.TrimSpace(*req.RegionID);action:=strings.TrimSpace(*req.ActionType);path:=strings.TrimSpace(*req.DominantPath);playerID,err:=uuid.Parse(strings.TrimSpace(*req.PlayerID));if region==""||len(region)>50||action==""||len(action)>30||len(path)>30||err!=nil||math.IsNaN(*req.DaoPulseStrength)||math.IsInf(*req.DaoPulseStrength,0)||*req.DaoPulseStrength<0||*req.DaoPulseStrength>1e9{c.JSON(http.StatusBadRequest,gin.H{"error":"Dữ liệu dấu vết không hợp lệ"});return};expires,err:=time.Parse(time.RFC3339,*req.ExpiresAt);if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Hạn dùng phải theo RFC3339"});return}
	var storedID string;if err:=db.DB.QueryRowContext(c.Request.Context(),`UPDATE world_residual_traces SET region_id=$1,x=$2,y=$3,player_id=$4,action_type=$5,dominant_path=$6,dao_pulse_strength=$7,expires_at=$8 WHERE id=$9 RETURNING id::text`,region,*req.X,*req.Y,playerID,action,path,*req.DaoPulseStrength,expires,traceID).Scan(&storedID);err!=nil{if err==sql.ErrNoRows{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy dấu vết"})}else{c.JSON(http.StatusBadRequest,gin.H{"error":"Không thể lưu dấu vết; hãy kiểm tra khu vực và người chơi"})};return}
	auditlog.Log(db.DB,auditlog.Entry{PlayerID:&adminID,Action:"admin.update_residual_trace",EntityType:"world_residual_traces",EntityID:storedID,RequestSummary:fmt.Sprintf("region=%s xy=%d,%d player=%s action=%s path=%s power=%g",region,*req.X,*req.Y,playerID,action,path,*req.DaoPulseStrength),IPAddress:c.ClientIP(),Success:true});c.JSON(http.StatusOK,gin.H{"message":"Đã cập nhật dấu vết thế giới"})
}

func AdminDeleteResidualTrace(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};traceID,err:=uuid.Parse(c.Param("id"));if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"ID dấu vết không hợp lệ"});return};var req struct{Reason string `json:"reason"`;ConfirmationTarget string `json:"confirmation_target"`};if err:=c.ShouldBindJSON(&req);err!=nil||strings.TrimSpace(req.Reason)==""||strings.TrimSpace(req.ConfirmationTarget)!=c.Request.URL.Path{c.JSON(http.StatusBadRequest,gin.H{"error":"Cần lý do và xác nhận đúng dấu vết cần xóa"});return};result,err:=db.DB.ExecContext(c.Request.Context(),`DELETE FROM world_residual_traces WHERE id=$1`,traceID);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xóa dấu vết"});return};if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy dấu vết"});return};auditlog.Log(db.DB,auditlog.Entry{PlayerID:&adminID,Action:"admin.delete_residual_trace",EntityType:"world_residual_traces",EntityID:traceID.String(),RequestSummary:"reason="+req.Reason,IPAddress:c.ClientIP(),Success:true});c.JSON(http.StatusOK,gin.H{"message":"Đã xóa dấu vết"})
}

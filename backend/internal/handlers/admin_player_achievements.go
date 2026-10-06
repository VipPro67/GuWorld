package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListPlayerAchievements(c *gin.Context) {
	if _,ok:=requireAdmin(c);!ok{return};playerID,err:=uuid.Parse(c.Param("id"));if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"ID người chơi không hợp lệ"});return}
	rows,err:=db.DB.QueryContext(c.Request.Context(),`SELECT player_id,achievement_id,selected_title,selected_frame,unlocked_at FROM player_achievements WHERE player_id=$1 ORDER BY unlocked_at DESC,achievement_id`,playerID);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tải thành tựu"});return};defer rows.Close();data:=make([]gin.H,0)
	for rows.Next(){var pID,achievementID,title,frame string;var unlocked time.Time;if err:=rows.Scan(&pID,&achievementID,&title,&frame,&unlocked);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc thành tựu"});return};data=append(data,gin.H{"player_id":pID,"achievement_id":achievementID,"selected_title":title,"selected_frame":frame,"unlocked_at":unlocked})};if err:=rows.Err();err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc thành tựu"});return};c.JSON(http.StatusOK,gin.H{"data":data,"total":len(data),"available":achievementRegistry})
}

func AdminAddPlayerAchievement(c *gin.Context){
	adminID,ok:=requireAdmin(c);if !ok{return};playerID,err:=uuid.Parse(c.Param("id"));if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"ID người chơi không hợp lệ"});return};var req struct{AchievementID string `json:"achievement_id"`;Reason string `json:"reason"`;ConfirmationTarget string `json:"confirmation_target"`};if err:=c.ShouldBindJSON(&req);err!=nil||strings.TrimSpace(req.Reason)==""||strings.TrimSpace(req.ConfirmationTarget)!=c.Request.URL.Path{c.JSON(http.StatusBadRequest,gin.H{"error":"Cần lý do và xác nhận cấp thành tựu"});return};achievementID:=strings.TrimSpace(req.AchievementID);known:=false;for _,definition:=range achievementRegistry{if definition.ID==achievementID{known=true;break}};if !known{c.JSON(http.StatusBadRequest,gin.H{"error":"Mã thành tựu không có trong registry"});return}
	result,err:=db.DB.ExecContext(c.Request.Context(),`INSERT INTO player_achievements(player_id,achievement_id) VALUES($1,$2) ON CONFLICT(player_id,achievement_id) DO NOTHING`,playerID,achievementID);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể cấp thành tựu"});return};n,_:=result.RowsAffected();if n==0{c.JSON(http.StatusConflict,gin.H{"error":"Người chơi đã có thành tựu này"});return};auditlog.Log(db.DB,auditlog.Entry{PlayerID:&adminID,Action:"admin.add_player_achievement",EntityType:"player_achievements",EntityID:playerID.String()+":"+achievementID,RequestSummary:fmt.Sprintf("achievement=%s reason=%s",achievementID,req.Reason),IPAddress:c.ClientIP(),Success:true});c.JSON(http.StatusOK,gin.H{"message":"Đã cấp thành tựu"})
}

func AdminUpdatePlayerAchievement(c *gin.Context){
	adminID,ok:=requireAdmin(c);if !ok{return};playerID,err:=uuid.Parse(c.Param("id"));if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"ID người chơi không hợp lệ"});return};achievementID:=strings.TrimSpace(c.Param("achievement_id"));var req struct{SelectedTitle *string `json:"selected_title"`;SelectedFrame *string `json:"selected_frame"`;UnlockedAt *string `json:"unlocked_at"`};if err:=c.ShouldBindJSON(&req);err!=nil||req.SelectedTitle==nil||req.SelectedFrame==nil||req.UnlockedAt==nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Cần đủ danh hiệu, khung và thời điểm mở khóa"});return};if len(*req.SelectedTitle)>100||len(*req.SelectedFrame)>50{c.JSON(http.StatusBadRequest,gin.H{"error":"Danh hiệu tối đa 100 ký tự, khung tối đa 50 ký tự"});return};unlocked,err:=time.Parse(time.RFC3339,*req.UnlockedAt);if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"Thời điểm mở khóa không hợp lệ"});return}
	var storedID string;if err:=db.DB.QueryRowContext(c.Request.Context(),`UPDATE player_achievements SET selected_title=$1,selected_frame=$2,unlocked_at=$3 WHERE player_id=$4 AND achievement_id=$5 RETURNING achievement_id`,*req.SelectedTitle,*req.SelectedFrame,unlocked,playerID,achievementID).Scan(&storedID);err!=nil{if err==sql.ErrNoRows{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy thành tựu của người chơi"})}else{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể cập nhật thành tựu"})};return};auditlog.Log(db.DB,auditlog.Entry{PlayerID:&adminID,Action:"admin.update_player_achievement",EntityType:"player_achievements",EntityID:playerID.String()+":"+storedID,RequestSummary:fmt.Sprintf("title=%q frame=%q unlocked_at=%s",*req.SelectedTitle,*req.SelectedFrame,unlocked.Format(time.RFC3339)),IPAddress:c.ClientIP(),Success:true});c.JSON(http.StatusOK,gin.H{"message":"Đã cập nhật thành tựu"})
}

func AdminDeletePlayerAchievement(c *gin.Context){
	adminID,ok:=requireAdmin(c);if !ok{return};playerID,err:=uuid.Parse(c.Param("id"));if err!=nil{c.JSON(http.StatusBadRequest,gin.H{"error":"ID người chơi không hợp lệ"});return};achievementID:=strings.TrimSpace(c.Param("achievement_id"));var req struct{Reason string `json:"reason"`;ConfirmationTarget string `json:"confirmation_target"`};if err:=c.ShouldBindJSON(&req);err!=nil||strings.TrimSpace(req.Reason)==""||strings.TrimSpace(req.ConfirmationTarget)!=c.Request.URL.Path{c.JSON(http.StatusBadRequest,gin.H{"error":"Cần lý do và xác nhận đúng thành tựu cần thu hồi"});return};result,err:=db.DB.ExecContext(c.Request.Context(),`DELETE FROM player_achievements WHERE player_id=$1 AND achievement_id=$2`,playerID,achievementID);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể thu hồi thành tựu"});return};if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy thành tựu"});return};auditlog.Log(db.DB,auditlog.Entry{PlayerID:&adminID,Action:"admin.delete_player_achievement",EntityType:"player_achievements",EntityID:playerID.String()+":"+achievementID,RequestSummary:fmt.Sprintf("reason=%s",req.Reason),IPAddress:c.ClientIP(),Success:true});c.JSON(http.StatusOK,gin.H{"message":"Đã thu hồi thành tựu"})
}

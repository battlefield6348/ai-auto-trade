package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleJobsStatus(c *gin.Context) {
	s.jobMu.Lock()
	history := make([]jobRun, len(s.jobHistory))
	copy(history, s.jobHistory)
	lastAuto := s.lastAutoRun
	s.jobMu.Unlock()

	loc := taipeiLocation()
	var lastAutoStr string
	if !lastAuto.IsZero() {
		lastAutoStr = lastAuto.In(loc).Format(time.RFC3339)
	}

	var latest map[string]interface{}
	if len(history) > 0 {
		latest = jobRunToMap(history[len(history)-1], loc)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"last_auto_run": latest,
		"last_auto_end": lastAutoStr,
	})
}

func (s *Server) handleJobsHistory(c *gin.Context) {
	s.jobMu.Lock()
	history := make([]jobRun, len(s.jobHistory))
	copy(history, s.jobHistory)
	s.jobMu.Unlock()

	loc := taipeiLocation()
	data := make([]map[string]interface{}, len(history))
	for i, j := range history {
		// Reverse order
		data[len(history)-1-i] = jobRunToMap(j, loc)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

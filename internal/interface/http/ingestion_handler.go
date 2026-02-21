package httpapi

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleIngestionBackfill(c *gin.Context) {
	var body struct {
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		RunAnalysis bool   `json:"run_analysis"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	start, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_date", "error_code": errCodeBadRequest})
		return
	}
	end, err := time.Parse("2006-01-02", body.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_date", "error_code": errCodeBadRequest})
		return
	}

	triggeredBy := currentUserID(c)
	job := jobRun{
		Kind:        "backfill",
		TriggeredBy: triggeredBy,
		Start:       time.Now(),
		AnalysisOn:  body.RunAnalysis,
	}
	
	log.Printf("[Backfill] Starting sync from %s to %s, triggered by %s", body.StartDate, body.EndDate, triggeredBy)
	
	total := 0
	succ := 0
	fail := 0
	var failures []backfillFailure

	curr := start
	for !curr.After(end) {
		// Ingestion
		if err := s.generateDailyPricesStrict(c.Request.Context(), curr); err != nil {
			log.Printf("[Backfill] Ingestion failed for %s: %v", curr.Format("2006-01-02"), err)
			failures = append(failures, backfillFailure{
				TradeDate: curr.Format("2006-01-02"),
				Stage:     "ingestion",
				Reason:    err.Error(),
			})
		} else {
			// Analysis
			if body.RunAnalysis {
				summary, err := s.runAnalysisForDate(c.Request.Context(), curr)
				if err != nil {
					log.Printf("[Backfill] Analysis failed for %s: %v", curr.Format("2006-01-02"), err)
					failures = append(failures, backfillFailure{
						TradeDate: curr.Format("2006-01-02"),
						Stage:     "analysis",
						Reason:    err.Error(),
					})
				} else {
					total += summary.total
					succ += summary.success
					fail += summary.failure
				}
			}
		}
		curr = curr.AddDate(0, 0, 1)
	}

	job.End = time.Now()
	job.AnalysisTotal = total
	job.AnalysisSucc = succ
	job.AnalysisFail = fail
	job.AnalysisOK = (fail == 0)
	job.IngestionOK = (len(failures) == 0)
	job.Failures = failures
	s.recordJob(job)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Backfill from %s to %s completed", body.StartDate, body.EndDate),
		"summary": job,
	})
}

func (s *Server) handleIngestionDaily(c *gin.Context) {
	var body struct {
		TradeDate   string `json:"trade_date"`
		RunAnalysis bool   `json:"run_analysis"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	tradeDate := time.Now()
	var err error
	if body.TradeDate != "" {
		tradeDate, err = time.Parse("2006-01-02", body.TradeDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid trade_date", "error_code": errCodeBadRequest})
			return
		}
	}

	triggeredBy := currentUserID(c)
	job := jobRun{
		Kind:        "daily_manual",
		TriggeredBy: triggeredBy,
		Start:       time.Now(),
		AnalysisOn:  body.RunAnalysis,
	}
	
	if err := s.generateDailyPrices(c.Request.Context(), tradeDate); err != nil {
		job.IngestionOK = false
		job.IngestionErr = err.Error()
	} else {
		job.IngestionOK = true
		if body.RunAnalysis {
			summary, err := s.runAnalysisForDate(c.Request.Context(), tradeDate)
			if err != nil {
				job.AnalysisOK = false
				job.AnalysisErr = err.Error()
			} else {
				job.AnalysisOK = true
				job.AnalysisTotal = summary.total
				job.AnalysisSucc = summary.success
				job.AnalysisFail = summary.failure
			}
		}
	}
	
	job.End = time.Now()
	s.recordJob(job)

	c.JSON(http.StatusOK, gin.H{
		"success": job.IngestionOK && (!body.RunAnalysis || job.AnalysisOK),
		"summary": job,
	})
}

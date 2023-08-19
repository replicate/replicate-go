package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/sync/errgroup"
)

type GetPrediction struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	URLs    struct {
		Get    string `json:"get"`
		Cancel string `json:"cancel"`
	} `json:"urls"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	Input       struct {
		Prompt string `json:"prompt"`
	} `json:"input"`
	Output  []string    `json:"output"`
	Error   interface{} `json:"error"`
	Logs    string      `json:"logs"`
	Metrics struct {
		PredictTime float64 `json:"predict_time"`
	} `json:"metrics"`
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/webhook", func(c echo.Context) error {
		// save a generated image file to a directory
		var output GetPrediction
		if err := c.Bind(&output); err != nil {
			return c.String(http.StatusInternalServerError, "bind error")
		}

		if err := DownloadImages(context.Background(), output.Output, "../data"); err != nil {
			return c.JSON(400, struct {
				Message string `json:"message"`
			}{Message: "faild to download image file."})
		}

		return c.JSON(http.StatusOK, struct {
			Status string
		}{Status: "OK"})
	})

	e.Logger.Fatal(e.Start(":" + "8080"))
}

func DownloadImages(ctx context.Context, output []string, dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return fmt.Errorf("%v is not a directory path", dirPath)
	}
	g, _ := errgroup.WithContext(ctx)

	for _, o := range output {
		o := o
		g.Go(func() error {
			res, err := http.DefaultClient.Get(o)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			url, err := url.Parse(o)
			out, err := os.Create(filepath.Join(dirPath, filepath.Dir(url.Path[1:])+filepath.Base(url.Path)))

			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, res.Body); err != nil {
				return err
			}
			return nil
		})
	}
	return g.Wait()
}

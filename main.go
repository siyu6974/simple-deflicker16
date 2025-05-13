package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/disintegration/imaging"
)

type picture struct {
	currentPath         string
	targetPath          string
	currentRgbHistogram rgbHistogram
	targetRgbHistogram  rgbHistogram
}

func main() {
	//Initial console output
	printInfo()
	//Read parameters from console
	config = collectConfigInformation()
	//Initialize Window from config and start GUI
	initalizeWindow()
	window.Main()
	os.Exit(0)
}

func runDeflickering() error {

	//Prepare
	configError := validateConfigInformation()
	if configError != nil {
		return configError
	}
	clear()
	runtime.GOMAXPROCS(config.threads)
	pictures, picturesError := readDirectory(config.sourceDirectory, config.destinationDirectory)
	if picturesError != nil {
		return picturesError
	}
	progress := createProgressBars(len(pictures))
	progress.container.Start()

	//Analyze and create Histograms
	var analyzeError error
	pictures, analyzeError = forEveryPicture(pictures, progress.bars["analyze"], config.threads, func(pic picture) (picture, error) {
		img, err := imaging.Open(pic.currentPath)
		if err != nil {
			return pic, errors.New(pic.currentPath + " | " + err.Error())
		}
		pic.currentRgbHistogram = generateRgbHistogramFromImage(img)
		return pic, nil
	})
	if analyzeError != nil {
		progress.container.Stop()
		return analyzeError
	}

	//Calculate global or rolling average
	if config.rollingAverage < 1 {
		var averageRgbHistogram rgbHistogram
	// Use histSize for loops instead of hardcoded 256
	histLen := len(averageRgbHistogram.r) // Should be histSize
		for i := range pictures {
		for j := 0; j < histLen; j++ {
				averageRgbHistogram.r[j] += pictures[i].currentRgbHistogram.r[j]
				averageRgbHistogram.g[j] += pictures[i].currentRgbHistogram.g[j]
				averageRgbHistogram.b[j] += pictures[i].currentRgbHistogram.b[j]
			}
		}
	for i := 0; i < histLen; i++ {
		if len(pictures) > 0 { // Avoid division by zero
			averageRgbHistogram.r[i] /= uint32(len(pictures))
			averageRgbHistogram.g[i] /= uint32(len(pictures))
			averageRgbHistogram.b[i] /= uint32(len(pictures))
		}
		}
		for i := range pictures {
			pictures[i].targetRgbHistogram = averageRgbHistogram
		}
	} else {
		for i := range pictures {
			var averageRgbHistogram rgbHistogram
			var start = i - config.rollingAverage
			if start < 0 {
				start = 0
			}
			var end = i + config.rollingAverage
			if end > len(pictures)-1 {
				end = len(pictures) - 1
			}
		// Use histSize for loops instead of hardcoded 256
		histLen := len(averageRgbHistogram.r) // Should be histSize
		for idx := start; idx <= end; idx++ { // Renamed loop variable to avoid conflict
			for j := 0; j < histLen; j++ {
				averageRgbHistogram.r[j] += pictures[idx].currentRgbHistogram.r[j]
				averageRgbHistogram.g[j] += pictures[idx].currentRgbHistogram.g[j]
				averageRgbHistogram.b[j] += pictures[idx].currentRgbHistogram.b[j]
				}
			}
		if (end - start + 1) > 0 { // Avoid division by zero
			for j := 0; j < histLen; j++ { // Renamed loop variable
				averageRgbHistogram.r[j] /= uint32(end - start + 1)
				averageRgbHistogram.g[j] /= uint32(end - start + 1)
				averageRgbHistogram.b[j] /= uint32(end - start + 1)
			}
			}
			pictures[i].targetRgbHistogram = averageRgbHistogram
		}
	}

	var adjustError error
	pictures, adjustError = forEveryPicture(pictures, progress.bars["adjust"], config.threads, func(pic picture) (picture, error) {
	var img, errOpen = imaging.Open(pic.currentPath)
	if errOpen != nil {
		return pic, errors.New("Error opening image for adjust: " + pic.currentPath + " | " + errOpen.Error())
	}
		lut := generateRgbLutFromRgbHistograms(pic.currentRgbHistogram, pic.targetRgbHistogram)
		img = applyRgbLutToImage(img, lut)

	// Ensure targetPath has .png extension
	currentExt := filepath.Ext(pic.targetPath)
	pic.targetPath = strings.TrimSuffix(pic.targetPath, currentExt) + ".png"

	// Save as PNG, removing JPEG specific options
	err := imaging.Save(img, pic.targetPath) // Removed JPEGQuality and PNGCompressionLevel
		if err != nil {
		return pic, errors.New("Error saving image: " + pic.targetPath + " | " + err.Error())
		}
		return pic, nil
	})
	if adjustError != nil {
		progress.container.Stop()
		return adjustError
	}
	progress.container.Stop()
	clear()
	fmt.Printf("Saved %v pictures into %v", len(pictures),config.destinationDirectory)
	return nil
}

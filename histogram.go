package main

import (
	"image"
	"image/color"
)

const histSize = 65536

type lut [histSize]uint16
type rgbLut struct {
	r lut
	g lut
	b lut
}
type histogram [histSize]uint32
type rgbHistogram struct {
	r histogram
	g histogram
	b histogram
}

func generateRgbHistogramFromImage(input image.Image) rgbHistogram {
	var rgbHistogram rgbHistogram
	bounds := input.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := input.At(x, y).RGBA()
			// RGBA() returns values in [0, 0xffff], so no shift is needed for 16-bit.
			rgbHistogram.r[r]++
			rgbHistogram.g[g]++
			rgbHistogram.b[b]++
		}
	}
	return rgbHistogram
}

func convertToCumulativeRgbHistogram(input rgbHistogram) rgbHistogram {
	var targetRgbHistogram rgbHistogram
	targetRgbHistogram.r[0] = input.r[0]
	targetRgbHistogram.g[0] = input.g[0]
	targetRgbHistogram.b[0] = input.b[0]
	for i := 1; i < histSize; i++ {
		targetRgbHistogram.r[i] = targetRgbHistogram.r[i-1] + input.r[i]
		targetRgbHistogram.g[i] = targetRgbHistogram.g[i-1] + input.g[i]
		targetRgbHistogram.b[i] = targetRgbHistogram.b[i-1] + input.b[i]
	}
	return targetRgbHistogram
}

func generateRgbLutFromRgbHistograms(current rgbHistogram, target rgbHistogram) rgbLut {
	currentCumulativeRgbHistogram := convertToCumulativeRgbHistogram(current)
	targetCumulativeRgbHistogram := convertToCumulativeRgbHistogram(target)
	var ratio [3]float64
	ratio[0] = float64(currentCumulativeRgbHistogram.r[histSize-1]) / float64(targetCumulativeRgbHistogram.r[histSize-1])
	ratio[1] = float64(currentCumulativeRgbHistogram.g[histSize-1]) / float64(targetCumulativeRgbHistogram.g[histSize-1])
	ratio[2] = float64(currentCumulativeRgbHistogram.b[histSize-1]) / float64(targetCumulativeRgbHistogram.b[histSize-1])

	for i := 0; i < histSize; i++ {
		targetCumulativeRgbHistogram.r[i] = uint32(0.5 + float64(targetCumulativeRgbHistogram.r[i])*ratio[0])
		targetCumulativeRgbHistogram.g[i] = uint32(0.5 + float64(targetCumulativeRgbHistogram.g[i])*ratio[1])
		targetCumulativeRgbHistogram.b[i] = uint32(0.5 + float64(targetCumulativeRgbHistogram.b[i])*ratio[2])
	}

	//Generate LUT
	var lut rgbLut
	var p [3]uint16 // Changed from uint8 to uint16
	for i := 0; i < histSize; i++ {
		// Ensure p values don't exceed histSize-1
		for p[0] < histSize-1 && targetCumulativeRgbHistogram.r[p[0]] < currentCumulativeRgbHistogram.r[i] {
			p[0]++
		}
		for p[1] < histSize-1 && targetCumulativeRgbHistogram.g[p[1]] < currentCumulativeRgbHistogram.g[i] {
			p[1]++
		}
		for p[2] < histSize-1 && targetCumulativeRgbHistogram.b[p[2]] < currentCumulativeRgbHistogram.b[i] {
			p[2]++
		}
		lut.r[i] = p[0]
		lut.g[i] = p[1]
		lut.b[i] = p[2]
	}
	return lut
}

func applyRgbLutToImage(input image.Image, lut rgbLut) image.Image {
	bounds := input.Bounds()
	// Create a new NRGBA64 image to support 16-bit color depth.
	// imaging.Clone will create an image of the same type, so if input is NRGBA, output is NRGBA
	// We need to ensure the output is NRGBA64 if we want 16-bit output.
	// However, the problem asks to "work with the new 16-bit LUT", which might mean the LUT
	// is 16-bit, but the image processing pipeline might still output 8-bit images.
	// For now, let's assume the function should try to output an NRGBA64 image.
	// A more robust solution would be to check the input image type and decide.

	dst := image.NewNRGBA64(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := input.At(x, y).RGBA() // r,g,b,a are uint32 in [0, 0xffff]

			// Apply 16-bit LUT. r, g, b are effectively uint16.
			newR := lut.r[r]
			newG := lut.g[g]
			newB := lut.b[b]

			dst.SetNRGBA64(x, y, color.NRGBA64{R: newR, G: newG, B: newB, A: uint16(a)})
		}
	}
	return dst
}

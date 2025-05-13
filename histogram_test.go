package main

import (
	"image"
	"image/color"
	"testing"
)

// Helper function to create a sample image for testing.
// Creates a 1xN image where N is the number of pixels.
func createSampleImage(pixels []color.Color) image.Image {
	width := len(pixels)
	height := 1
	img := image.NewNRGBA64(image.Rect(0, 0, width, height))
	for x, pxColor := range pixels {
		img.Set(x, 0, pxColor)
	}
	return img
}

// Helper function to check if two histograms are equal.
func histogramsAreEqual(h1, h2 rgbHistogram, t *testing.T) bool {
	for i := 0; i < histSize; i++ {
		if h1.r[i] != h2.r[i] || h1.g[i] != h2.g[i] || h1.b[i] != h2.b[i] {
			// For debugging, print the first differing index and values
			// t.Logf("Histograms differ at index %d:\n H1_R: %d, H2_R: %d\n H1_G: %d, H2_G: %d\n H1_B: %d, H2_B: %d",
			// 	i, h1.r[i], h2.r[i], h1.g[i], h2.g[i], h1.b[i], h2.b[i])
			return false
		}
	}
	return true
}

func TestGenerateRgbHistogramFromImage_BrightnessCutoff(t *testing.T) {
	// Define pixel colors (using uint16 for 16-bit depth)
	// Brightness = (R+G+B)/3
	// Black: (0,0,0), Brightness = 0
	black := color.NRGBA64{R: 0, G: 0, B: 0, A: 0xffff}
	// Gray (8-bit 30): (30*257, 30*257, 30*257), Brightness = 30*257 = 7710
	grayDark := color.NRGBA64{R: uint16(30 * 257), G: uint16(30 * 257), B: uint16(30 * 257), A: 0xffff}
	// Gray (8-bit 100): (100*257, 100*257, 100*257), Brightness = 100*257 = 25700
	grayMedium := color.NRGBA64{R: uint16(100 * 257), G: uint16(100 * 257), B: uint16(100 * 257), A: 0xffff}
	// Bright (8-bit 200): (200*257, 200*257, 200*257), Brightness = 200*257 = 51400
	bright := color.NRGBA64{R: uint16(200 * 257), G: uint16(200 * 257), B: uint16(200 * 257), A: 0xffff}

	samplePixels := []color.Color{black, grayDark, grayMedium, bright}
	testImage := createSampleImage(samplePixels)

	tests := []struct {
		name            string
		minBrightness   uint32 // Effective 16-bit brightness level
		expectedHist    rgbHistogram
		pixelsToCount   []color.Color // For easier expected histogram construction
	}{
		{
			name:          "Cutoff below medium gray (cutoff 8-bit 50)",
			minBrightness: uint32(50 * 257), // Brightness = 12850
			// Expected: grayMedium and bright pixels are counted. black and grayDark are ignored.
			// grayDark (brightness 7710) < 12850
			// grayMedium (brightness 25700) >= 12850
			// bright (brightness 51400) >= 12850
			pixelsToCount: []color.Color{grayMedium, bright},
		},
		{
			name:          "Cutoff very high (cutoff 8-bit 220)",
			minBrightness: uint32(220 * 257), // Brightness = 56540
			// Expected: No pixels counted
			pixelsToCount: []color.Color{},
		},
		{
			name:          "No cutoff (minBrightness 0)",
			minBrightness: 0,
			// Expected: All pixels counted
			pixelsToCount: []color.Color{black, grayDark, grayMedium, bright},
		},
		{
			name:          "Cutoff exactly at a pixel's brightness (cutoff 8-bit 30)",
			minBrightness: uint32(30*257 + 1), // Cutoff just above grayDark's brightness
			// Expected: grayMedium and bright are counted.
			// black (0) < 7711
			// grayDark (7710) < 7711
			pixelsToCount: []color.Color{grayMedium, bright},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct expected histogram
			var expected rgbHistogram
			for _, pxColor := range tt.pixelsToCount {
				r, g, b, _ := pxColor.RGBA() // Returns uint32 in [0, 0xffff]
				expected.r[r]++
				expected.g[g]++
				expected.b[b]++
			}

			actualHist := generateRgbHistogramFromImage(testImage, tt.minBrightness)

			if !histogramsAreEqual(actualHist, expected, t) {
				// More detailed logging if histogramsAreEqual doesn't log enough
				// This is tricky because large arrays are hard to print concisely.
				// For now, rely on histogramsAreEqual to indicate inequality.
				// We can add specific checks for key pixel values if needed.
				t.Errorf("Test Case '%s': generateRgbHistogramFromImage() returned unexpected histogram.\nExpected pixel counts based on brightness cutoff not met.", tt.name)
				// Example of specific value check (if black was expected to be counted or not)
				// if tt.minBrightness == 0 {
				// 	if actualHist.r[0] == 0 { t.Error("Black pixel was not counted when minBrightness was 0") }
				// } else if tt.minBrightness > 0 && actualHist.r[0] > 0 {
				//  t.Error("Black pixel was counted when it should have been cut off")
				// }
			}
		})
	}
}

func TestGenerateRgbLutFromRgbHistograms_EmptyTarget(t *testing.T) {
	var currentHist rgbHistogram
	// Populate currentHist with some data, e.g., a peak at mid-intensity (128 * 257)
	// For simplicity, let's make it a single point with 100 counts
	midIntensityVal := uint16(128 * 257) // Example intensity
	currentHist.r[midIntensityVal] = 100
	currentHist.g[midIntensityVal] = 100
	currentHist.b[midIntensityVal] = 100

	// targetHist is all zeros by default for a new rgbHistogram{}
	var targetHist rgbHistogram

	lut := generateRgbLutFromRgbHistograms(currentHist, targetHist)

	// Assert that the LUT is an identity LUT
	identity := true
	for i := 0; i < histSize; i++ {
		if lut.r[i] != uint16(i) || lut.g[i] != uint16(i) || lut.b[i] != uint16(i) {
			identity = false
			t.Errorf("TestGenerateRgbLutFromRgbHistograms_EmptyTarget: LUT is not identity at index %d.\nExpected R[%d]=%d, Got %d\nExpected G[%d]=%d, Got %d\nExpected B[%d]=%d, Got %d",
				i, i, i, lut.r[i], i, i, lut.g[i], i, i, lut.b[i])
			break // No need to check further if one mismatch is found
		}
	}

	if !identity {
		t.Error("TestGenerateRgbLutFromRgbHistograms_EmptyTarget: Resulting LUT is not an identity LUT when target histogram is empty.")
	}
}

// Optional Test: TestApplyRgbLutToImage_WithCutoffEffect
// This test is more complex to assert correctly.
// A simple start is to check if LUTs generated with and without cutoff (on current) are different.
func TestLutDifference_WithAndWithoutCutoff(t *testing.T) {
	black := color.NRGBA64{R: 0, G: 0, B: 0, A: 0xffff}
	grayMedium := color.NRGBA64{R: uint16(100 * 257), G: uint16(100 * 257), B: uint16(100 * 257), A: 0xffff}
	bright := color.NRGBA64{R: uint16(200 * 257), G: uint16(200 * 257), B: uint16(200 * 257), A: 0xffff}
	
	samplePixels := []color.Color{black, grayMedium, bright}
	testImage := createSampleImage(samplePixels)

	// Target histogram: let's make it uniform for simplicity, or an average of the image itself.
	// For this test, a simple non-empty target is enough.
	var targetHist rgbHistogram
	targetHist.r[uint16(150*257)] = 10 // Some arbitrary non-empty target
	targetHist.g[uint16(150*257)] = 10
	targetHist.b[uint16(150*257)] = 10


	// LUT1: No brightness cutoff for current histogram
	currentHistNoCutoff := generateRgbHistogramFromImage(testImage, 0)
	lut1 := generateRgbLutFromRgbHistograms(currentHistNoCutoff, targetHist)

	// LUT2: With brightness cutoff for current histogram (cutoff black pixels)
	minBrightnessCutoffBlack := uint32(10 * 257) // Cutoff brightness > 0, e.g., 8-bit 10
	currentHistWithCutoff := generateRgbHistogramFromImage(testImage, minBrightnessCutoffBlack)
	
	// Ensure currentHistWithCutoff is different from currentHistNoCutoff due to the black pixel
	if histogramsAreEqual(currentHistNoCutoff, currentHistWithCutoff, t) && currentHistNoCutoff.r[0] > 0 {
		t.Fatal("TestLutDifference: currentHistWithCutoff should be different from currentHistNoCutoff due to brightness cut-off, but they are identical.")
	}
	if currentHistWithCutoff.r[0] != 0 {
		t.Errorf("TestLutDifference: Black pixel (value 0) should have been cut from currentHistWithCutoff, but count is %d", currentHistWithCutoff.r[0])
	}


	lut2 := generateRgbLutFromRgbHistograms(currentHistWithCutoff, targetHist)

	// Assert that lut1 and lut2 are different
	lutsAreDifferent := false
	for i := 0; i < histSize; i++ {
		if lut1.r[i] != lut2.r[i] || lut1.g[i] != lut2.g[i] || lut1.b[i] != lut2.b[i] {
			lutsAreDifferent = true
			// t.Logf("LUTs differ at index %d: LUT1(R:%d,G:%d,B:%d), LUT2(R:%d,G:%d,B:%d)", i, lut1.r[i], lut1.g[i], lut1.b[i], lut2.r[i], lut2.g[i], lut2.b[i])
			break
		}
	}

	if !lutsAreDifferent {
		// This can happen if the target histogram is such that the cutoff makes no difference to the final LUT.
		// For example, if targetHist was also empty, both LUTs would be identity.
		// Or if the cut-off pixels didn't influence the LUT mapping for other pixels significantly.
		// This test is sensitive to the choice of targetHist and cutoff values.
		// A more robust check would be needed for a true integration test of applyRgbLutToImage.
		t.Error("TestLutDifference: lut1 (no cutoff) and lut2 (with cutoff) were expected to be different, but they are identical. This might indicate an issue or a specific test data interaction.")
	}
}

// Note: The `histSize` constant is defined in histogram.go (value 65536).
// The RGBA() method from image.Color returns uint32 components, but for 16-bit images (like NRGBA64),
// these values are effectively uint16s in the range [0, 0xFFFF].
// The brightness calculation (r+g+b)/3 will also be in this range.
// The `minBrightness` parameter in `generateRgbHistogramFromImage` is uint32,
// so direct comparison with scaled 16-bit brightness values is fine.

// To run tests: go test .
// To run with verbose output: go test -v .
// To run a specific test: go test -run TestName
// Example: go test -v -run TestGenerateRgbHistogramFromImage_BrightnessCutoff
// Example: go test -v -run TestGenerateRgbLutFromRgbHistograms_EmptyTarget
// Example: go test -v -run TestLutDifference_WithAndWithoutCutoff

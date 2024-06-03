package main

import (
	"fmt"
	"os/exec"
	"strings"

	"azul3d.org/engine/gfx"
	"azul3d.org/engine/gfx/window"
)

func gfxLoop(w window.Window, d gfx.Device) {
	defer w.Close()

	dev := d.Info()
	fmt.Println("Device Name:", dev.Name)
	fmt.Println("Device Vendor:", dev.Vendor)
	fmt.Println("OcclusionQuery =", dev.OcclusionQuery)
	fmt.Println("OcclusionQueryBits =", dev.OcclusionQueryBits)
	fmt.Println("MaxTextureSize =", dev.MaxTextureSize)
	fmt.Println("NPOT Textures =", dev.NPOT)
	fmt.Println("AlphaToCoverage =", dev.AlphaToCoverage)
	fmt.Println("DepthClamp =", dev.DepthClamp)

	fmt.Println(dev.GL)
	fmt.Println(dev.GLSL)
	fmt.Println("GLSL MaxVaryingFloats =", dev.GLSL.MaxVaryingFloats)
	fmt.Println("GLSL MaxVertexInputs =", dev.GLSL.MaxVertexInputs)
	fmt.Println("GLSL MaxFragmentInputs =", dev.GLSL.MaxFragmentInputs)

	fmt.Printf("%d Render-To-Texture MSAA Formats:\n", len(dev.RTTFormats.Samples))
	for i, sampleCount := range dev.RTTFormats.Samples {
		fmt.Printf("    %d. %dx MSAA\n", i+1, sampleCount)
	}

	fmt.Printf("%d Render-To-Texture Color Formats:\n", len(dev.RTTFormats.ColorFormats))
	for i, f := range dev.RTTFormats.ColorFormats {
		fmt.Printf("    %d. %+v\n", i+1, f)
	}
	fmt.Printf("%d Render-To-Texture Depth Formats:\n", len(dev.RTTFormats.DepthFormats))
	for i, f := range dev.RTTFormats.DepthFormats {
		fmt.Printf("    %d. %+v\n", i+1, f)
	}
	fmt.Printf("%d Render-To-Texture Stencil Formats:\n", len(dev.RTTFormats.StencilFormats))
	for i, f := range dev.RTTFormats.StencilFormats {
		fmt.Printf("    %d. %+v\n", i+1, f)
	}

	fmt.Println("OpenGL extensions:", dev.GL.Extensions)
}

func detectGPU() {
	props := window.NewProps()
	props.SetVisible(false)
	props.SetResizeRenderSync(false)
	window.Run(gfxLoop, props)
}

func detectGPU_Windows() {
	cmd := exec.Command("wmic", "path", "win32_VideoController", "get", "name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// The output contains column headers and possibly multiple lines, so we split it into lines
	lines := strings.Split(string(output), "\n")

	// The actual GPU name should be on the second line (index 1), if it exists
	if len(lines) > 1 {
		fmt.Println("GPU:", lines[1])
	} else {
		fmt.Println("Could not detect GPU")
	}
}

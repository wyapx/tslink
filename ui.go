package main

import (
	"fmt"
	"image"
	"image/color"
	"tslink/core"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type uiState struct {
	initState int
	nodeName  string
	peers     []core.LinkPeerConnectivityStatus
	history   []float64
	ready     bool
	list      layout.List
	logList   layout.List
	err       string
	logs      []string
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	var ops op.Ops
	state := &uiState{
		nodeName: "unknown",
	}
	state.logList.ScrollToEnd = true
	state.logList.Axis = layout.Vertical
	state.list.Axis = layout.Vertical

	go func() {
		for e := range core.Events {
			switch event := e.(type) {
			case *core.LinkInitEvent:
				state.initState = event.State
				if event.State == core.LinkInitReady {
					state.ready = true
					state.logs = state.logs[:0]
				}
			case *core.LogEvent:
				state.logs = append(state.logs, event.Message)
				if len(state.logs) > 100 {
					state.logs = state.logs[1:]
				}
			case *core.LinkErrorEvent:
				state.err = event.Error
			case *core.HostnameAssignedEvent:
				state.nodeName = event.Hostname
			case *core.LinkPeerConnectivityEvent:
				state.peers = event.PingResult
				var totalLat float64
				var count int
				for _, p := range event.PingResult {
					if p.Result != nil {
						totalLat += p.Result.LatencySeconds
						count++
					}
				}
				if count > 0 {
					state.history = append(state.history, totalLat/float64(count))
					if len(state.history) > 50 {
						state.history = state.history[1:]
					}
				}
			}
			window.Invalidate()
		}
	}()

	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if !state.ready {
				drawLoading(gtx, theme, state)
			} else {
				drawMain(gtx, theme, state)
			}
			e.Frame(gtx.Ops)
		}
	}
}

func drawLoading(gtx layout.Context, theme *material.Theme, state *uiState) layout.Dimensions {
	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions { // log painter
			return layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return state.logList.Layout(gtx, len(state.logs), func(gtx layout.Context, i int) layout.Dimensions {
						l := material.Caption(theme, state.logs[i])
						l.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 50}
						return l.Layout(gtx)
					})
				})
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if state.err != "" {
						return layout.Dimensions{}
					}
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Loader(theme).Layout(gtx)
					})
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					txt := "Initializing..."
					textColor := theme.Palette.Fg
					if state.err != "" {
						txt = state.err
						textColor = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
					} else {
						switch state.initState {
						case core.LinkInitFetchConfig:
							txt = "Fetching configuration..."
						case core.LinkInitConnectingTailscale:
							txt = "Connecting to Tailscale..."
						case core.LinkInitControlPlaneConnected:
							txt = "Control plane connected..."
						case core.LinkInitProgramSetup:
							txt = "Setting up program..."
						}
					}
					l := material.Body1(theme, txt)
					l.Color = textColor
					l.Alignment = text.Middle
					return l.Layout(gtx)
				}),
			)
		}),
	)
}

func drawMain(gtx layout.Context, theme *material.Theme, state *uiState) layout.Dimensions {
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := material.H4(theme, fmt.Sprintf("Node: %s", state.nodeName))
				return h.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.H6(theme, "Peers").Layout(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return state.list.Layout(gtx, len(state.peers), func(gtx layout.Context, i int) layout.Dimensions {
							p := state.peers[i]
							lat := "N/A"
							conn := "unknown"
							if p.Result != nil {
								lat = fmt.Sprintf("%.2fms", p.Result.LatencySeconds*1000)
								if p.Result.DERPRegionCode == "" {
									conn = "direct"
								} else {
									conn = fmt.Sprintf("derp(%s)", p.Result.DERPRegionCode)
								}
							}
							return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return material.Body1(theme, fmt.Sprintf("%s: %s (%s)", p.Target, lat, conn)).Layout(gtx)
							})
						})
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.H6(theme, "Overall Latency History").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return drawChart(gtx, state.history)
					}),
				)
			}),
		)
	})
}

func drawChart(gtx layout.Context, history []float64) layout.Dimensions {
	size := image.Pt(gtx.Constraints.Max.X, 150)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, color.NRGBA{R: 30, G: 30, B: 30, A: 255})

	if len(history) < 2 {
		return layout.Dimensions{Size: size}
	}

	var maxLat float64
	for _, h := range history {
		if h > maxLat {
			maxLat = h
		}
	}
	if maxLat == 0 {
		maxLat = 1
	}

	var path clip.Path
	path.Begin(gtx.Ops)

	w := float32(size.X)
	h := float32(size.Y)

	xStep := w / float32(len(history)-1)

	for i, lat := range history {
		x := float32(i) * xStep
		y := h - (float32(lat/maxLat) * h * 0.8) // Leave some space at top

		if i == 0 {
			path.MoveTo(f32.Pt(x, y))
		} else {
			path.LineTo(f32.Pt(x, y))
		}
	}

	paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 200, B: 0, A: 255}, clip.Stroke{
		Path:  path.End(),
		Width: float32(gtx.Dp(unit.Dp(2))),
	}.Op())

	return layout.Dimensions{Size: size}
}

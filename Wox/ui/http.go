package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/olahol/melody"
	"github.com/samber/lo"
	"net/http"
	"os"
	"strings"
	"time"
	"wox/plugin"
	"wox/resource"
	"wox/setting"
	"wox/ui/dto"
	"wox/util"
)

var m *melody.Melody
var uiConnected = false

type websocketMsgType string

const (
	WebsocketMsgTypeRequest  websocketMsgType = "WebsocketMsgTypeRequest"
	WebsocketMsgTypeResponse websocketMsgType = "WebsocketMsgTypeResponse"
)

type WebsocketMsg struct {
	Id      string
	Type    websocketMsgType
	Method  string
	Success bool
	Data    any
}

type RestResponse struct {
	Success bool
	Message string
	Data    any
}

func writeSuccessResponse(w http.ResponseWriter, data any) {
	d, marshalErr := json.Marshal(RestResponse{
		Success: true,
		Message: "",
		Data:    data,
	})
	if marshalErr != nil {
		writeErrorResponse(w, marshalErr.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(d)
}

func writeErrorResponse(w http.ResponseWriter, errMsg string) {
	d, _ := json.Marshal(RestResponse{
		Success: false,
		Message: errMsg,
		Data:    "",
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write(d)
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func serveAndWait(ctx context.Context, port int) {
	m = melody.New()
	m.Config.MaxMessageSize = 1024 * 1024 * 10 // 10MB
	m.Config.MessageBufferSize = 1024 * 1024   // 1MB

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeSuccessResponse(w, "Wox")
	})

	http.HandleFunc("/index.html", func(w http.ResponseWriter, r *http.Request) {
		fileContent, err := resource.GetReactFile(util.NewTraceContext(), "index.html")
		if err != nil {
			writeErrorResponse(w, err.Error())
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(fileContent)
	})

	http.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/assets/")
		fileContent, err := resource.GetReactFile(util.NewTraceContext(), "assets", path)
		if err != nil {
			writeErrorResponse(w, err.Error())
			return
		}

		contentType := "text/plain"
		if strings.HasSuffix(path, "js") {
			contentType = "text/javascript; charset=utf-8"
		}
		if strings.HasSuffix(path, "css") {
			contentType = "text/css"
		}

		w.Header().Set("Content-Type", contentType)
		w.Write(fileContent)
	})

	http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeErrorResponse(w, "id is empty")
			return
		}

		imagePath, ok := plugin.GetLocalImageMap(id)
		if !ok {
			writeErrorResponse(w, "imagePath is empty")
			return
		}

		if _, statErr := os.Stat(imagePath); os.IsNotExist(statErr) {
			w.Write([]byte("image not exist"))
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeFile(w, r, imagePath)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	http.HandleFunc("/theme", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)
		theme := GetUIManager().GetCurrentTheme(util.NewTraceContext())
		writeSuccessResponse(w, theme)
	})

	http.HandleFunc("/plugin/store", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)
		manifests := plugin.GetStoreManager().GetStorePluginManifests(util.NewTraceContext())
		var plugins = make([]dto.StorePlugin, len(manifests))
		copyErr := copier.Copy(&plugins, &manifests)
		if copyErr != nil {
			writeErrorResponse(w, copyErr.Error())
			return
		}

		for i, storePlugin := range plugins {
			isInstalled := lo.ContainsBy(plugin.GetPluginManager().GetPluginInstances(), func(item *plugin.Instance) bool {
				return item.Metadata.Id == storePlugin.Id
			})
			plugins[i].Icon = plugin.NewWoxImageUrl(manifests[i].IconUrl)
			plugins[i].IsInstalled = isInstalled
		}

		writeSuccessResponse(w, plugins)
	})

	http.HandleFunc("/plugin/installed", func(w http.ResponseWriter, r *http.Request) {
		defer util.GoRecover(util.NewTraceContext(), "get installed plugins")
		enableCors(w)

		getCtx := util.NewTraceContext()
		instances := plugin.GetPluginManager().GetPluginInstances()
		var plugins []dto.InstalledPlugin
		for _, instance := range instances {
			var installedPlugin dto.InstalledPlugin
			copyErr := copier.Copy(&installedPlugin, &instance.Metadata)
			if copyErr != nil {
				writeErrorResponse(w, copyErr.Error())
				return
			}

			logger.Debug(getCtx, fmt.Sprintf("get plugin setting: %s", instance.Metadata.Name))
			installedPlugin.SettingDefinitions = instance.Metadata.SettingDefinitions

			var definitionSettings = util.NewHashMap[string, string]()
			for _, item := range instance.Metadata.SettingDefinitions {
				settingValue := instance.API.GetSetting(getCtx, item.Value.GetKey())
				definitionSettings.Store(item.Value.GetKey(), settingValue)
			}
			installedPlugin.Settings = *instance.Setting
			//only return user pre-defined settings
			installedPlugin.Settings.Settings = definitionSettings

			iconImg, parseErr := plugin.ParseWoxImage(instance.Metadata.Icon)
			if parseErr == nil {
				installedPlugin.Icon = iconImg
			} else {
				installedPlugin.Icon = plugin.NewWoxImageBase64(`data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADAAAAAwCAYAAABXAvmHAAAACXBIWXMAAAsTAAALEwEAmpwYAAAELUlEQVR4nO3ZW2xTdRwH8JPgkxE1XuKFQUe73rb1IriNyYOJvoiALRszvhqffHBLJjEx8Q0TlRiN0RiNrPd27boLY1wUFAQHyquJiTIYpefay7au2yiJG1/zb6Kx/ZfS055T1mS/5JtzXpr+Pufy//9P/gyzURulXIHp28Rp7H5eY/OSc6bRiiPNN9tBQs4bDsFrbN5/AQ2JANO3qRgx9dZ76I2vwingvsQhgHUK2NPQCKeAuOw7Mf72B1hPCEZu9bBrWE8IRm6RH60nBFMNQA3Eh6kVzCzzyOVu5I+HUyvqApREkOZxe5bKR+kVdQFKIcgVLwW4usyrD1ACcSsXKwm4lYvVB1Ar4r7fAWeNCPLClgIcruBFVhRQK4Jc8Vwulj/WZRQqh4i8+X5d5glGaYCDBzp/WYQ5KsJ98JDqCEZJgIO/g53nM9BHpXxMEQHuXnURjFIA0vyOHxfQMiIVxBgW4FIRwSgBcLB3YPt+DrqwWDKGEA9Xz7tlES/9nkPHuQyeP5/By3/crh9gf3wNlpMpaENC2egDHFwHSiBurqL78hLaJlNoPZaCeSIJ01gSu68sqw/YF1uDeTKF5qBQUXR+DkNFiOgbg7BOSBTAOJrIw1QD7J1dzf+Jxs/LitbL4qhjsAAROjAA67hEAQzRBLovLSkPePX6an6U2eblqorWE4en7xCFsIxJFEAfkcoiZANeufo3tMMitnq4qkIArZMp2E+k4H29COEcQPuoSAH0YQm7prPKAMhjsMXFVpUmN4f2qTRsp+dgPTUH21SyJKJtRKQALcNiSYRswLNH46gmTW4W7SfSsP8w/x/AcjIN6/EkvEWPU9AxgNaIQAF0IRFdF7O1AZ75Lg65aXKxsJxKw35mngIQlOVYoiTCHBYogDZQiJANePrbm5AT8uhYT8/hubPzdwW0HU/BMpGA5yCNMIV4CrDdL6DzQrY6wFPfxFBpSPOkabLEuBeAzAOWcYIonCcCr/XDFOQpQLOPIBblA578OoZKQprfcXYeO88tVAwg80D7mARPL40wBjgKoPHy8gFPfHUD90q++Z8W8usauQAyD7RFJbiLEfv7YfBztQMe/3IW5bLFzeYb7/g5UzXANJZE64gIdw+N0Hu52gCPfTGLu2Wrm0XHhQw6Ly7WDDCOJmCOiNQq1r+vHy0etnrAo59fR6mQcZ40Tr7GlAIYogmYhgVqFUsQOjdbHeCRz66hONt8HLqmF9E1nVUcoI9IMIUIYpBGuOLyAQ9/eg3/jybAo/tyFrsuZVUD6MMSjEGeQvj2vgMwLz4gC7D5yAy7+cgMSLYHBbzw2xK6f1Uf0DIswhDgMeQsRPAae0QW4sGP/9zz0Cd/seRTcfeVpboCdCEReh+PIUeNiHWx7dtsDxQibF6mkQpFCHLONOYGvM1Hmm+obd+NYhqg/gG2aOxED6eh5gAAAABJRU5ErkJggg==`)
			}

			plugins = append(plugins, installedPlugin)
		}

		writeSuccessResponse(w, plugins)
	})

	http.HandleFunc("/theme/store", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)
		storeThemes := GetStoreManager().GetThemes()
		var themes = make([]dto.Theme, len(storeThemes))
		copyErr := copier.Copy(&themes, &storeThemes)
		if copyErr != nil {
			writeErrorResponse(w, copyErr.Error())
			return
		}

		for i, storeTheme := range themes {
			isInstalled := lo.ContainsBy(GetUIManager().GetAllThemes(ctx), func(item Theme) bool {
				return item.ThemeId == storeTheme.ThemeId
			})
			themes[i].IsInstalled = isInstalled
		}

		writeSuccessResponse(w, themes)
	})

	http.HandleFunc("/theme/installed", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)
		installedThemes := GetUIManager().GetAllThemes(ctx)
		var themes = make([]dto.Theme, len(installedThemes))
		copyErr := copier.Copy(&themes, &installedThemes)
		if copyErr != nil {
			writeErrorResponse(w, copyErr.Error())
			return
		}

		writeSuccessResponse(w, themes)
	})

	http.HandleFunc("/setting/wox", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)

		woxSetting := setting.GetSettingManager().GetWoxSetting(util.NewTraceContext())

		var settingDto dto.WoxSetting
		copyErr := copier.Copy(&settingDto, &woxSetting)
		if copyErr != nil {
			writeErrorResponse(w, copyErr.Error())
			return
		}

		settingDto.MainHotkey = woxSetting.MainHotkey.Get()
		settingDto.SelectionHotkey = woxSetting.SelectionHotkey.Get()
		settingDto.QueryHotkeys = woxSetting.QueryHotkeys.Get()

		writeSuccessResponse(w, settingDto)
	})

	http.HandleFunc("/setting/wox/update", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)

		type keyValuePair struct {
			Key   string
			Value string
		}

		decoder := json.NewDecoder(r.Body)
		var kv keyValuePair
		err := decoder.Decode(&kv)
		if err != nil {
			w.Header().Set("code", "500")
			w.Write([]byte(err.Error()))
			return
		}

		updateErr := setting.GetSettingManager().UpdateWoxSetting(util.NewTraceContext(), kv.Key, kv.Value)
		if updateErr != nil {
			w.Header().Set("code", "500")
			w.Write([]byte(updateErr.Error()))
			return
		}

		writeSuccessResponse(w, "")
	})

	http.HandleFunc("/open/url", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)

		url := r.URL.Query().Get("url")
		if url == "" {
			writeErrorResponse(w, "url is empty")
			return
		}

		util.ShellOpen(url)

		writeSuccessResponse(w, "")
	})

	m.HandleConnect(func(s *melody.Session) {
		if !uiConnected {
			uiConnected = true
			logger.Info(ctx, fmt.Sprintf("ui connected: %s", s.Request.RemoteAddr))

			util.Go(ctx, "post app start", func() {
				time.Sleep(time.Millisecond * 500) // wait for ui to be ready
				GetUIManager().PostAppStart(util.NewTraceContext())
			})
		}
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		ctxNew := util.NewTraceContext()

		logger.Info(ctxNew, fmt.Sprintf("got request from ui: %s", string(msg)))

		if strings.Contains(string(msg), string(WebsocketMsgTypeRequest)) {
			var request WebsocketMsg
			unmarshalErr := json.Unmarshal(msg, &request)
			if unmarshalErr != nil {
				logger.Error(ctxNew, fmt.Sprintf("failed to unmarshal websocket request: %s", unmarshalErr.Error()))
				return
			}
			util.Go(ctxNew, "handle ui query", func() {
				onUIRequest(ctxNew, request)
			})
		}
	})

	logger.Info(ctx, fmt.Sprintf("websocket server start at：ws://localhost:%d", port))
	err := http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil)
	if err != nil {
		logger.Error(ctx, fmt.Sprintf("failed to start server: %s", err.Error()))
	}
}

func requestUI(ctx context.Context, request WebsocketMsg) {
	request.Type = WebsocketMsgTypeRequest
	marshalData, marshalErr := json.Marshal(request)
	if marshalErr != nil {
		logger.Error(ctx, fmt.Sprintf("failed to marshal websocket request: %s", marshalErr.Error()))
		return
	}

	jsonData, _ := json.Marshal(request.Data)
	util.GetLogger().Info(ctx, fmt.Sprintf("[->UI] %s: %s", request.Method, jsonData))
	m.Broadcast(marshalData)
}

func responseUI(ctx context.Context, response WebsocketMsg) {
	response.Type = WebsocketMsgTypeResponse
	marshalData, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		logger.Error(ctx, fmt.Sprintf("failed to marshal websocket response: %s", marshalErr.Error()))
		return
	}
	m.Broadcast(marshalData)
}

func responseUISuccessWithData(ctx context.Context, request WebsocketMsg, data any) {
	responseUI(ctx, WebsocketMsg{
		Id:      request.Id,
		Type:    WebsocketMsgTypeResponse,
		Method:  request.Method,
		Success: true,
		Data:    data,
	})
}

func responseUISuccess(ctx context.Context, request WebsocketMsg) {
	responseUISuccessWithData(ctx, request, nil)
}

func responseUIError(ctx context.Context, request WebsocketMsg, errMsg string) {
	responseUI(ctx, WebsocketMsg{
		Id:      request.Id,
		Type:    WebsocketMsgTypeResponse,
		Method:  request.Method,
		Success: false,
		Data:    errMsg,
	})
}

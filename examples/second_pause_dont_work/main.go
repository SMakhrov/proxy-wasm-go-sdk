// Copyright 2020-2021 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

var original_headers = types.Headers{}

func main() {
	proxywasm.SetNewRootContext(newRootContext)
}

type rootContext struct {
	// You'd better embed the default root context
	// so that you don't need to reimplement all the methods by yourself.
	proxywasm.DefaultRootContext
}

func newRootContext(uint32) proxywasm.RootContext { return &rootContext{} }

// Override DefaultRootContext.
func (*rootContext) NewHttpContext(contextID uint32) proxywasm.HttpContext {
	return &httpHeadersBody{contextID: contextID}
}

type httpHeadersBody struct {
	// You'd better embed the default root context
	// so that you don't need to reimplement all the methods by yourself.
	proxywasm.DefaultHttpContext
	contextID uint32
}

// Override DefaultHttpContext.
func (ctx *httpHeadersBody) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	hs, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get request headers: %v", err)
	}
	original_headers = hs

	for _, h := range hs {
		proxywasm.LogInfof("request header --> %s: %s", h[0], h[1])
	}

	if _, err := proxywasm.DispatchHttpCall("web_service_canary", original_headers, "", nil,
		5000, callback1); err != nil {
		proxywasm.LogCriticalf("dispatch httpcall 1 failed: %v", err)
		return types.ActionContinue
	}
	return types.ActionPause
}

func callback1(numHeaders, bodySize, numTrailers int) {
	proxywasm.LogInfof("callback 1 was called")
	proxywasm.ResumeHttpRequest()
}

func (ctx *httpHeadersBody) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	if _, err := proxywasm.DispatchHttpCall("web_service", original_headers, "", nil,
		5000, callback2); err != nil {
		proxywasm.LogCriticalf("dispatch httpcall 2 failed: %v", err)
		return types.ActionContinue
	}
	return types.ActionPause // this is not working because proxywasm.ResumeHttpRequest() was called at callback1
}

func callback2(numHeaders, bodySize, numTrailers int) {
	proxywasm.LogInfof("callback 2 was called")
	body := "access forbidden"
	proxywasm.LogInfo(body)
	if err := proxywasm.SendHttpResponse(403, types.Headers{
		{"powered-by", "proxy-wasm-go-sdk!!"},
	}, []byte(body)); err != nil {
		proxywasm.LogErrorf("failed to send local response: %v", err)
		proxywasm.ResumeHttpRequest()
	}
}

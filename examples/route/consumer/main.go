/**
 * Tencent is pleased to support the open source community by making polaris-go available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

var (
	namespace     string
	service       string
	port          int64
	selfNamespace string
	selfService   string
)

func initArgs() {
	flag.StringVar(&namespace, "namespace", "default", "namespace")
	flag.StringVar(&service, "service", "RouteEchoServer", "service")
	flag.StringVar(&selfNamespace, "selfNamespace", "default", "selfNamespace")
	flag.StringVar(&selfService, "selfService", "", "selfService")
}

type PolarisConsumer struct {
	consumer  api.ConsumerAPI
	namespace string
	service   string
}

func (svr *PolarisConsumer) Run() {
	svr.runWebServer()
}

func (svr *PolarisConsumer) runWebServer() {
	http.HandleFunc("/echo", func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("start to invoke getOneInstance operation")
		getOneRequest := &api.GetOneInstanceRequest{}
		getOneRequest.Namespace = namespace
		getOneRequest.Service = service
		getOneRequest.SourceService = &model.ServiceInfo{
			Service:   selfService,
			Namespace: selfNamespace,
			Metadata:  convertHeaders(r.Header),
		}
		oneInstResp, err := svr.consumer.GetOneInstance(getOneRequest)
		if nil != err {
			log.Printf("[error] fail to getOneInstance, err is %v", err)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(fmt.Sprintf("fail to getOneInstance, err is %v", err)))
			return
		}
		instance := oneInstResp.GetInstance()
		if nil != instance {
			log.Printf("instance getOneInstance is %s:%d", instance.GetHost(), instance.GetPort())
		}

		resp, err := http.Get(fmt.Sprintf("http://%s:%d/echo", instance.GetHost(), instance.GetPort()))
		if err != nil {
			log.Printf("[error] send request to %s:%d fail : %s", instance.GetHost(), instance.GetPort(), err)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(fmt.Sprintf("send request to %s:%d fail : %s", instance.GetHost(), instance.GetPort(), err)))
			return
		}

		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("read resp from %s:%d fail : %s", instance.GetHost(), instance.GetPort(), err)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(fmt.Sprintf("read resp from %s:%d fail : %s", instance.GetHost(), instance.GetPort(), err)))
			return
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(data)
	})

	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", 18080), nil); err != nil {
		log.Fatalf("[ERROR]fail to run webServer, err is %v", err)
	}
}

func main() {
	initArgs()
	flag.Parse()
	if len(namespace) == 0 || len(service) == 0 {
		log.Print("namespace and service are required")
		return
	}
	consumer, err := api.NewConsumerAPI()
	if nil != err {
		log.Fatalf("fail to create consumerAPI, err is %v", err)
	}
	defer consumer.Destroy()

	svr := &PolarisConsumer{
		consumer:  consumer,
		namespace: namespace,
		service:   service,
	}

	svr.Run()

}

func convertHeaders(header map[string][]string) map[string]string {
	meta := make(map[string]string)
	for k, v := range header {
		meta[strings.ToLower(k)] = v[0]
	}

	return meta
}

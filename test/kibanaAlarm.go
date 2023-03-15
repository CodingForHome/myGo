package mq

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/url"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	"gitlab.bambu-lab.com/devops/mayday/mayday-consumer/utils"
)

//go:embed tpl/alarm-nameSpace-service.tpl
var kibanaAlarmTpl string

const (
	kibanaUrl  = `https://kibana.bambu-lab.com/s/log/app/discover#/?`
	nameSpaceG = `(filters:!(),refreshInterval:(pause:!t,value:0),time:(from:'%s',to:'%s'))`
	nameSpaceA = `(columns:!(trace,namespace,content,service),filters:!(),index:'log-collector*',interval:auto,query:(language:kuery,query:'namespace : "%s" and level : "%s"'),sort:!(!('@timestamp',desc)))`
	traceG     = `(filters:!(),refreshInterval:(pause:!t,value:0),time:(from:now-15m,to:now))`
	traceA     = `(columns:!(trace,namespace,content,service),filters:!(),index:'log-collector*',interval:auto,query:(language:kuery,query:'trace : %s'),sort:!(!('@timestamp',desc)))`
)

const (
	chatId = "oc_39a4db077de4360b3faed052f8c69a8b"
)

var AlarmChan = make(chan AlarmInfo, 10)

// AlarmInfo 用于报警信息
type AlarmInfo struct {
	NameSpace string
	Service   string
	Level     string
	Trace     string
	AlarmTime int64
}

// AlarmService 用于聚合信息
type AlarmService struct {
	Total     int64
	FirstTime int64
	LastTime  int64
	Trace     []string
}
type AlarmNameSpace struct {
	Total         int64
	FirstTime     int64
	LastTime      int64
	Level         string
	AlarmServices map[string]*AlarmService
}

// 用于构建消息卡片
type service struct {
	ServiceName string
	Total       int64
	Link        []string
}
type kibanaAlarmLark struct {
	NameSpace      string
	Total          int64
	LevelNameSpace string
	Services       []service
	CardSendTime   string
}

func NewKibanaAlarmLark(nameSpace string, total int64, levelNameSpace string, services []service, cardSendTime string) *kibanaAlarmLark {
	return &kibanaAlarmLark{
		NameSpace:      nameSpace,
		Total:          total,
		LevelNameSpace: levelNameSpace,
		Services:       services,
		CardSendTime:   cardSendTime,
	}
}

func (a kibanaAlarmLark) BuildCard() (string, error) {
	tpl, err := template.New("kibanaAlarm").Parse(kibanaAlarmTpl)
	if err != nil {
		return "", errors.Wrapf(err, "parse tpl error")
	}

	var b bytes.Buffer
	err = tpl.Execute(&b, a)
	if err != nil {
		return "", errors.Wrapf(err, "execute error:%+v", a)
	}

	return b.String(), nil
}

type Aggregator struct {
	data      map[string]*AlarmNameSpace // 聚合map
	alarmChan chan AlarmInfo             // 收集alarm信息的channel
	ticker    *time.Ticker               // 定时器
}

func NewAggregator(alarmChan chan AlarmInfo) *Aggregator {
	return &Aggregator{
		data:      make(map[string]*AlarmNameSpace),
		alarmChan: alarmChan,
	}
}

func (a *Aggregator) Start(d time.Duration) {
	a.ticker = time.NewTicker(d)
	for {
		select {
		case <-a.ticker.C:
			a.sendToFeiShu()
		case alarmInfo := <-a.alarmChan:
			a.aggAlarmInfo(alarmInfo)
		}
	}
}
func (a *Aggregator) sendToFeiShu() {
	for nameSpace, aggNameSpaceInfo := range a.data {
		// 构建kibana链接
		levelSpaceUrl := BuildKibanaUrl(formatTime(aggNameSpaceInfo.FirstTime), formatTime(aggNameSpaceInfo.LastTime), nameSpace, aggNameSpaceInfo.Level, "")
		var services []service
		for serviceName, serviceData := range aggNameSpaceInfo.AlarmServices {
			newService := service{
				ServiceName: serviceName,
				Total:       serviceData.Total,
			}
			for _, s := range serviceData.Trace {
				if len(newService.Link) >= 5 {
					break
				}
				// trace为空就基于nameSpace+level
				if s == "" {
					newService.Link = append(newService.Link, BuildKibanaUrl(formatTime(serviceData.FirstTime), formatTime(serviceData.LastTime), nameSpace, aggNameSpaceInfo.Level, ""))
				} else {
					newService.Link = append(newService.Link, BuildKibanaUrl("", "", "", "", s))
				}
			}
			services = append(services, newService)
		}
		// 进行报警并初始化
		kibanaAlarmLark := NewKibanaAlarmLark(nameSpace, aggNameSpaceInfo.Total, levelSpaceUrl, services, formatTime(time.Now().Unix()))
		card, buildCardErr := kibanaAlarmLark.BuildCard()
		if buildCardErr != nil {
			logx.Errorf("Error building card: %v", buildCardErr)
		}
		// 发送飞书报警
		_, sendCardErr := utils.AlarmLarkBot.SendCard(context.TODO(), chatId, card)
		if sendCardErr != nil {
			logx.Errorf("Error sending card: %v", sendCardErr)
		}
	}
	// 初始化聚合信息
	a.data = make(map[string]*AlarmNameSpace)

}

func (a *Aggregator) aggAlarmInfo(alarmInfo AlarmInfo) {
	_, ok := a.data[alarmInfo.NameSpace]
	if !ok {
		// 初始化nameSpace聚合信息
		a.data[alarmInfo.NameSpace] = &AlarmNameSpace{
			Total:         0,
			Level:         alarmInfo.Level,
			FirstTime:     alarmInfo.AlarmTime,
			LastTime:      alarmInfo.AlarmTime,
			AlarmServices: make(map[string]*AlarmService),
		}
	}
	aggNameSpaceInfo := a.data[alarmInfo.NameSpace]
	// 进行聚合
	aggNameSpaceInfo.Total++
	aggNameSpaceInfo.LastTime = alarmInfo.AlarmTime
	_, ok = aggNameSpaceInfo.AlarmServices[alarmInfo.Service]
	if !ok {
		// 初始化service聚合信息
		newAlarmService := &AlarmService{
			Total:     0,
			FirstTime: alarmInfo.AlarmTime,
			LastTime:  alarmInfo.AlarmTime,
		}
		aggNameSpaceInfo.AlarmServices[alarmInfo.Service] = newAlarmService
	}
	alarmServiceInfo := aggNameSpaceInfo.AlarmServices[alarmInfo.Service]
	// 更新service信息
	alarmServiceInfo.Total++
	alarmServiceInfo.LastTime = alarmInfo.AlarmTime
	alarmServiceInfo.Trace = append(alarmServiceInfo.Trace, alarmInfo.Trace)

}

func formatTime(t int64) string {
	return time.Unix(t, 0).Format("2006-01-02T15:04:05.000Z")
}

func BuildKibanaUrl(firstTime, lastTime, nameSpace, level, traceId string) string {
	params := url.Values{}
	if traceId != "" {
		params.Add("_g", traceG)
		params.Add("_a", fmt.Sprintf(traceA, traceId))
	} else {
		params.Add("_g", fmt.Sprintf(nameSpaceG, firstTime, lastTime))
		params.Add("_a", fmt.Sprintf(nameSpaceA, nameSpace, level))
	}
	newUrl := kibanaUrl + params.Encode()
	return newUrl
}

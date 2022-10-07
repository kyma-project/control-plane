package logger

import "code.cloudfoundry.org/lager"

type BrokerLogger struct {
	target lager.Logger
}

func (b *BrokerLogger) RegisterSink(sink lager.Sink) {
	//TODO implement me
	panic("implement me")
}

func (b *BrokerLogger) Session(task string, data ...lager.Data) lager.Logger {
	//TODO implement me
	panic("implement me")
}

func (b *BrokerLogger) SessionName() string {
	//TODO implement me
	panic("implement me")
}

func (b *BrokerLogger) Debug(action string, data ...lager.Data) {
	//TODO implement me
	panic("implement me")
}

func (b *BrokerLogger) Info(action string, data ...lager.Data) {
	//TODO implement me
	panic("implement me")
}

func (b *BrokerLogger) Error(action string, err error, data ...lager.Data) {
	//TODO implement me
	panic("implement me")
}

func (b *BrokerLogger) Fatal(action string, err error, data ...lager.Data) {
	//TODO implement me
	panic("implement me")
}

func (b *BrokerLogger) WithData(data lager.Data) lager.Logger {
	//TODO implement me
	panic("implement me")
	b.target.
}


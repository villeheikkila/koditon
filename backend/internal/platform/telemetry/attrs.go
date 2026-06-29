package telemetry

import "go.opentelemetry.io/otel/attribute"

type attrBuilder struct{}

var Attrs attrBuilder

func (attrBuilder) OperationID(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_id", value)
}

func (attrBuilder) OperationSummary(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_summary", value)
}

func (attrBuilder) OperationPath(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_path", value)
}

func (attrBuilder) OperationMethod(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_method", value)
}

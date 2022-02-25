/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interpreter_test

import (
	"testing"
	"time"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/require"
)

func TestInterpreterTracing(t *testing.T) {

	t.Parallel()

	t.Run("composite tracing", func(t *testing.T) {
		storage := interpreter.NewInMemoryStorage()

		elaboration := sema.NewElaboration()
		elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

		traceOps := make([]string, 0)
		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Elaboration: elaboration,
			},
			utils.TestLocation,
			interpreter.WithOnRecordTraceHandler(
				func(inter *interpreter.Interpreter,
					operationName string,
					duration time.Duration,
					logs []opentracing.LogRecord) {
					traceOps = append(traceOps, operationName)
				},
			),
			interpreter.WithStorage(storage),
			interpreter.WithTracingEnabled(true),
		)
		require.NoError(t, err)

		owner := common.Address{0x1}

		value := newTestCompositeValue(inter, owner)

		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "composite.construct")

		cloned := value.Clone(inter)
		require.NotNil(t, cloned)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "composite.clone")

		cloned.DeepRemove(inter)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "composite.deepRemove")

		array := interpreter.NewArrayValue(
			inter,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.Address{},
			value,
		)
		require.NotNil(t, array)
		require.Equal(t, len(traceOps), 5)
		require.Equal(t, traceOps[3], "composite.transfer")
		require.Equal(t, traceOps[4], "array.construct")
	})
}

/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package sema

type RuntimeTypeConstructor struct {
	Name        string
	Value       *FunctionType
	Description string
}

var OptionalTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var VariableSizedArrayTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var ConstantSizedArrayTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "size",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var DictionaryTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "key",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "value",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

var CompositeTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "identifier",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

var InterfaceTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "identifier",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

var FunctionTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "parameters",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{Type: MetaType}),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "return",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var RestrictedTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "typeID",
			TypeAnnotation: NewTypeAnnotation(&OptionalType{StringType}),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "restrictions",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{Type: StringType}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

var ReferenceTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "authorized",
			TypeAnnotation: NewTypeAnnotation(BoolType),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var CapabilityTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

func RuntimeTypeConstructors() []*RuntimeTypeConstructor {
	var functions = []*RuntimeTypeConstructor{
		{
			"OptionalType",
			OptionalTypeFunctionType,
			"Creates a run-time type representing an optional version of the given run-time type.",
		},

		{
			"VariableSizedArrayType",
			VariableSizedArrayTypeFunctionType,
			"Creates a run-time type representing a variable-sized array type of the given run-time type.",
		},

		{
			"ConstantSizedArrayType",
			ConstantSizedArrayTypeFunctionType,
			"Creates a run-time type representing a constant-sized array type of the given run-time type with the specifized size.",
		},

		{
			"DictionaryType",
			DictionaryTypeFunctionType,
			"Creates a run-time type representing a dictionary type of the given run-time key and value types. Returns nil if the key type is not a valid dictionary key.",
		},

		{
			"CompositeType",
			CompositeTypeFunctionType,
			"Creates a run-time type representing the composite type associated with the given type identifier. Returns nil if the identifier does not correspond to any composite type.",
		},

		{
			"InterfaceType",
			InterfaceTypeFunctionType,
			"Creates a run-time type representing the interface type associated with the given type identifier. Returns nil if the identifier does not correspond to any interface type.",
		},

		{
			"FunctionType",
			FunctionTypeFunctionType,
			"Creates a run-time type representing a function type associated with the given parameters and return type.",
		},

		{
			"ReferenceType",
			ReferenceTypeFunctionType,
			"Creates a run-time type representing a reference type of the given type, with authorization provided by the first argument.",
		},

		{
			"RestrictedType",
			RestrictedTypeFunctionType,
			"Creates a run-time type representing a restricted type of the first argument, restricted by the interface identifiers in the second argument. Returns nil if the restriction is not valid.",
		},

		{
			"CapabilityType",
			CapabilityTypeFunctionType,
			"Creates a run-time type representing a capability type of the given reference type. Returns nil if the type is not a reference.",
		},
	}

	return functions
}

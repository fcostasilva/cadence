package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/sema"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckCastingIntLiteralToIntegerType(t *testing.T) {

	for _, integerType := range sema.AllIntegerTypes {

		t.Run(integerType.String(), func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x = 1 as %s
                    `,
					integerType,
				),
			)

			require.NoError(t, err)

			assert.Equal(t,
				integerType,
				checker.GlobalValues["x"].Type,
			)

			assert.NotEmpty(t, checker.Elaboration.CastingTargetTypes)
		})
	}
}

func TestCheckInvalidCastingIntLiteralToString(t *testing.T) {

	_, err := ParseAndCheck(t, `
      let x = 1 as String
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckCastingIntLiteralToAnyStruct(t *testing.T) {

	checker, err := ParseAndCheck(t, `
      let x = 1 as AnyStruct
    `)

	require.NoError(t, err)

	assert.Equal(t,
		&sema.AnyStructType{},
		checker.GlobalValues["x"].Type,
	)

	assert.NotEmpty(t, checker.Elaboration.CastingTargetTypes)
}

func TestCheckCastingResourceToAnyResource(t *testing.T) {

	checker, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let r <- create R()
          let x <- r as @AnyResource
          destroy x
      }
    `)

	require.NoError(t, err)

	assert.NotEmpty(t, checker.Elaboration.CastingTargetTypes)
}

func TestCheckCastingArrayLiteral(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun zipOf3(a: [AnyStruct; 3], b: [Int; 3]): [[AnyStruct; 2]; 3] {
          return [
              [a[0], b[0]] as [AnyStruct; 2],
              [a[1], b[1]] as [AnyStruct; 2],
              [a[2], b[2]] as [AnyStruct; 2]
          ]
      }
    `)

	require.NoError(t, err)
}

func TestCheckCastResourceType(t *testing.T) {

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1, I2} <- create R()
                  let r2 <- r as @R{I2}
                `,
			)

			require.NoError(t, err)

			r2Type := checker.GlobalValues["r2"].Type

			require.IsType(t,
				&sema.RestrictedType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{I2}? {
                      let r: @R{I1, I2} <- create R()
                      if let r2 <- r as? @R{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @R{I1, I2}
                `,
			)

			require.NoError(t, err)

			r2Type := checker.GlobalValues["r2"].Type

			require.IsType(t,
				&sema.RestrictedType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{I1, I2}? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @R{I1, I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R1{I} <- create R1()
                  let r2 <- r as @R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R2{I}? {
                      let r: @R1{I} <- create R1()
                      if let r2 <- r as? @R2{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @R{I}
                `,
			)

			require.NoError(t, err)

			r2Type := checker.GlobalValues["r2"].Type

			require.IsType(t,
				&sema.RestrictedType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{I}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @R{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R1 <- create R1()
                  let r2 <- r as @R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R2{I}? {
                      let r: @R1 <- create R1()
                      if let r2 <- r as? @R2{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyResource -> conforming restricted type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @R{RI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{RI}? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @R{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> conforming restricted type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R{RI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{RI}? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> non-conforming restricted type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{RI}? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[2])

		})
	})

	// Supertype: Resource (unrestricted)

	t.Run("restricted type -> unrestricted type: same resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I} <- create R()
                  let r2 <- r as @R
                `,
			)

			require.NoError(t, err)

			r2Type := checker.GlobalValues["r2"].Type

			require.IsType(t,
				&sema.CompositeType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @R{I} <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}

          resource T: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R{I} <- create R()
                  let t <- r as @T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @T? {
                      let r: @R{I} <- create R()
                      if let t <- r as? @T {
                          return <-t
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyResource -> conforming resource", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> non-conforming resource", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyResource -> unrestricted type", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @R
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: restricted AnyResource

	t.Run("resource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const types = `
          resource interface RI {}

          // NOTE: R does not conform to RI
          resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @AnyResource{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{RI}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @AnyResource{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

	})

	t.Run("resource -> restricted AnyResource with conformance restriction", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @AnyResource{RI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{RI}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @AnyResource{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with conformance in restriction", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I} <- create R()
                  let r2 <- r as @AnyResource{I}
                `,
			)

			require.NoError(t, err)

			iType := checker.GlobalTypes["I"].Type

			require.IsType(t, &sema.InterfaceType{}, iType)

			r2Type := checker.GlobalValues["r2"].Type

			require.IsType(t,
				&sema.RestrictedType{
					Type: &sema.AnyResourceType{},
					Restrictions: []*sema.InterfaceType{
						iType.(*sema.InterfaceType),
					},
				},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I}? {
                      let r: @R{I} <- create R()
                      if let r2 <- r as? @AnyResource{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with conformance not in restriction", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @AnyResource{I2}
                `,
			)

			require.NoError(t, err)

			i2Type := checker.GlobalTypes["I2"].Type

			require.IsType(t, &sema.InterfaceType{}, i2Type)

			r2Type := checker.GlobalValues["r2"].Type

			require.IsType(t,
				&sema.RestrictedType{
					Type: &sema.AnyResourceType{},
					Restrictions: []*sema.InterfaceType{
						i2Type.(*sema.InterfaceType),
					},
				},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I2}? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @AnyResource{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I2}? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource: fewer restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1, I2} <- create R()
                  let r2 <- r as @AnyResource{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I2}? {
                      let r: @AnyResource{I1, I2} <- create R()
                      if let r2 <- r as? @AnyResource{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource: more restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1} <- create R()
                  let r2 <- r as @AnyResource{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I1, I2}? {
                      let r: @AnyResource{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I1, I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1} <- create R()
                  let r2 <- r as @AnyResource{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I1, I2}? {
                      let r: @AnyResource{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I1, I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyResource -> restricted AnyResource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @AnyResource{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I}? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @AnyResource{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: AnyResource

	t.Run("restricted type -> AnyResource", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @AnyResource {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> AnyResource", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1} <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r: @AnyResource{I1} <- create R()
                      if let r2 <- r as? @AnyResource {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> AnyResource", func(t *testing.T) {

		const types = `
           resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {
			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r <- create R()
                      if let r2 <- r as? @AnyResource {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})
}

func TestCheckCastStructType(t *testing.T) {

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1, I2} = S()
                  let s2 = s as S{I2}
                `,
			)

			require.NoError(t, err)

			s2Type := checker.GlobalValues["s2"].Type

			require.IsType(t,
				&sema.RestrictedType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1, I2} = S()
                  let s2 = s as? S{I2}
                `,
			)

			require.NoError(t, err)

			s2Type := checker.GlobalValues["s2"].Type

			require.IsType(t,
				&sema.OptionalType{
					Type: &sema.RestrictedType{},
				},
				s2Type,
			)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as S{I1, I2}
                `,
			)

			require.NoError(t, err)

			s2Type := checker.GlobalValues["s2"].Type

			require.IsType(t,
				&sema.RestrictedType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? S{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S1{I} = S1()
                  let s2 = s as S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S1{I} = S1()
                  let s2 = s as? S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as S{I}
                `,
			)

			require.NoError(t, err)

			s2Type := checker.GlobalValues["s2"].Type

			require.IsType(t,
				&sema.RestrictedType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? S{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S1 = S1()
                  let s2 = s as S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                   let s: S1 = S1()
                   let s2 = s as? S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyStruct -> conforming restricted type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as S{SI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? S{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> conforming restricted type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S{SI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> non-conforming restricted type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
		})
	})

	// Supertype: Struct (unrestricted)

	t.Run("restricted type -> unrestricted type: same struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as S
                `,
			)

			require.NoError(t, err)

			s2Type := checker.GlobalValues["s2"].Type

			require.IsType(t,
				&sema.CompositeType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as? S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}

          struct T: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: T{I} = S()
                  let t = s as T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: T{I} = S()
                  let t = s as? T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyStruct -> conforming struct", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> non-conforming struct", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyStruct -> unrestricted type", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? S
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: restricted AnyStruct

	t.Run("struct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const types = `
          struct interface SI {}

          // NOTE: S does not conform to SI
          struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as AnyStruct{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? AnyStruct{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

	})

	t.Run("struct -> restricted AnyStruct with conformance restriction", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as AnyStruct{SI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? AnyStruct{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with conformance in restriction", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as AnyStruct{I}
                `,
			)

			require.NoError(t, err)

			iType := checker.GlobalTypes["I"].Type

			require.IsType(t, &sema.InterfaceType{}, iType)

			s2Type := checker.GlobalValues["s2"].Type

			require.IsType(t,
				&sema.RestrictedType{
					Type: &sema.AnyStructType{},
					Restrictions: []*sema.InterfaceType{
						iType.(*sema.InterfaceType),
					},
				},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as? AnyStruct{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with conformance not in restriction", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as AnyStruct{I2}
                `,
			)

			require.NoError(t, err)

			i2Type := checker.GlobalTypes["I2"].Type

			require.IsType(t, &sema.InterfaceType{}, i2Type)

			s2Type := checker.GlobalValues["s2"].Type

			require.IsType(t,
				&sema.RestrictedType{
					Type: &sema.AnyStructType{},
					Restrictions: []*sema.InterfaceType{
						i2Type.(*sema.InterfaceType),
					},
				},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as AnyStruct{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? AnyStruct{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct: fewer restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1, I2} = S()
                  let s2 = s as AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1, I2} = S()
                  let s2 = s as? AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct: more restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as AnyStruct{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as? AnyStruct{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as AnyStruct{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as? AnyStruct{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyStruct -> restricted AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as AnyStruct{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? AnyStruct{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: AnyStruct

	t.Run("restricted type -> AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> AnyStruct", func(t *testing.T) {

		const types = `
           struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {
			_, err := ParseAndCheck(t,
				types+`
                  let s = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})
}

func TestCheckReferenceTypeSubTyping(t *testing.T) {

	t.Run("resource", func(t *testing.T) {

		for _, ty := range []string{
			"R",
			"R{I}",
			"AnyResource",
			"AnyResource{I}",
			"Any",
			"Any{I}",
		} {

			t.Run(fmt.Sprintf("auth to non-auth: %s", ty), func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(`
                          resource interface I {}

                          resource R: I {}

                          let r <- create R()
                          let ref = &r as auth &%[1]s
                          let ref2 = ref as &%[1]s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run(fmt.Sprintf("non-auth to auth: %s", ty), func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(`
                          resource interface I {}

                          resource R: I {}

                          let r <- create R()
                          let ref = &r as &%[1]s
                          let ref2 = ref as auth &%[1]s
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		}

	})

	t.Run("struct", func(t *testing.T) {

		for _, ty := range []string{
			"S",
			"S{I}",
			"AnyStruct",
			"AnyStruct{I}",
			"Any",
			"Any{I}",
		} {
			t.Run(fmt.Sprintf("auth to non-auth: %s", ty), func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(`
                          struct interface I {}

                          struct S: I {}

                          let s = S()
                          let ref = &s as auth &%[1]s
                          let ref2 = ref as &%[1]s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run(fmt.Sprintf("non-auth to auth: %s", ty), func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let s = S()
                          let ref = &s as &%[1]s
                          let ref2 = ref as auth &%[1]s
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		}

	})
}

func TestCheckCastAuthorizedResourceReferenceType(t *testing.T) {

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &R{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &R{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{I1, I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}

          let x <- create R1()
          let r = &x as auth &R1{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          let x <- create R()
          let r = &x as auth &R
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}

          let x <- create R1()
          let r = &x as auth &R1
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyResource -> conforming restricted type", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          resource R: RI {}

          let x <- create R()
          let r = &x as auth &AnyResource{RI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{RI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{RI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyResource -> conforming restricted type", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          resource R: RI {}

          let x <- create R()
          let r = &x as auth &AnyResource
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{RI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> non-conforming restricted type", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          resource R {}

          let x <- create R()
          let r = &x as auth &AnyResource{RI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
		})
	})

	// Supertype: Resource (unrestricted)

	t.Run("restricted type -> unrestricted type: same resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          let x <- create R()
          let r = &x as auth &R{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          resource T: I {}

          let x <- create R()
          let r = &x as auth &R{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = r as &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = r as? &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyResource -> conforming resource", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          resource R: RI {}

          let x <- create R()
          let r = &x as auth &AnyResource{RI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> non-conforming resource", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          resource R {}

          let x <- create R()
          let r = &x as auth &AnyResource{RI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyResource -> unrestricted type", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          resource R: RI {}

          let x <- create R()
          let r = &x as auth &AnyResource
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: restricted AnyResource

	t.Run("resource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          // NOTE: R does not conform to RI
          resource R {}

          let x <- create R()
          let r = &x as auth &R
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("resource -> restricted AnyResource with conformance restriction", func(t *testing.T) {

		const setup = `
          resource interface RI {}

          resource R: RI {}

          let x <- create R()
          let r = &x as auth &R
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{RI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{RI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with conformance in restriction", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          let x <- create R()
          let r = &x as auth &R{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with conformance not in restriction", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &R{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}

          let x <- create R()
          let r = &x as auth &R{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource: fewer restrictions", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &AnyResource{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource: more restrictions", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &AnyResource{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}

          let x <- create R()
          let r = &x as auth &AnyResource{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyResource -> restricted AnyResource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          let x <- create R()
          let r = &x as auth &AnyResource
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: AnyResource

	t.Run("restricted type -> AnyResource", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &R{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> AnyResource", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &AnyResource{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> AnyResource", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &R
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &AnyResource
                `,
			)

			require.NoError(t, err)
		})
	})
}

func TestCheckCastAuthorizedStructReferenceType(t *testing.T) {

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &S{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &S{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{I1, I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}

          let x = S1()
          let s = &x as auth &S1{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          let x = S()
          let s = &x as auth &S

        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}

          let x = S1()
          let s = &x as auth &S1
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyStruct -> conforming restricted type", func(t *testing.T) {

		const setup = `
          struct interface SI {}

          struct S: SI {}

          let x = S()
          let s = &x as auth &AnyStruct{SI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{SI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyStruct -> conforming restricted type", func(t *testing.T) {

		const setup = `
          struct interface SI {}

          struct S: SI {}

          let x = S()
          let s = &x as auth &AnyStruct
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> non-conforming restricted type", func(t *testing.T) {

		const setup = `
          struct interface SI {}

          struct S {}

          let x = S()
          let s = &x as auth &AnyStruct{SI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
		})
	})

	// Supertype: Struct (unrestricted)

	t.Run("restricted type -> unrestricted type: same struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          let x = S()
          let s = &x as auth &S{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          struct T: I {}

          let x = S()
          let s = &x as auth &S{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = s as &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = s as? &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyStruct -> conforming struct", func(t *testing.T) {

		const setup = `
          struct interface RI {}

          struct S: RI {}

          let x = S()
          let s = &x as auth &AnyStruct{RI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> non-conforming struct", func(t *testing.T) {

		const setup = `
          struct interface RI {}

          struct S {}

          let x = S()
          let s = &x as auth &AnyStruct{RI}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyStruct -> unrestricted type", func(t *testing.T) {

		const setup = `
          struct interface SI {}

          struct S: SI {}

          let x = S()
          let s = &x as auth &AnyStruct
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: restricted AnyStruct

	t.Run("struct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const setup = `
          struct interface SI {}

          // NOTE: S does not conform to SI
          struct S {}

          let x = S()
          let s = &x as auth &S
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("struct -> restricted AnyStruct with conformance restriction", func(t *testing.T) {

		const setup = `
          struct interface SI {}

          struct S: SI {}

          let x = S()
          let s = &x as auth &S
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{SI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with conformance in restriction", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          let x = S()
          let s = &x as auth &S{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with conformance not in restriction", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &S{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}

          let x = S()
          let s = &x as auth &S{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct: fewer restrictions", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &AnyStruct{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct: more restrictions", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &AnyStruct{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}

          let x = S()
          let s = &x as auth &AnyStruct{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyStruct -> restricted AnyStruct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          let x = S()
          let s = &x as auth &AnyStruct
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: AnyStruct

	t.Run("restricted type -> AnyStruct", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &S{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> AnyStruct", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &AnyStruct{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> AnyStruct", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &S
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})
}

func TestCheckCastUnauthorizedResourceReferenceType(t *testing.T) {

	for name, op := range map[string]string{
		"static":  "as",
		"dynamic": "as?",
	} {

		t.Run(name, func(t *testing.T) {

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &R{I1, I2}
                          let r2 = r %s &R{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &R{I1}
                          let r2 = r %s &R{I1, I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R1: I {}

                          resource R2: I {}

                          let x <- create R1()
                          let r = &x as &R1{I}
                          let r2 = r %s &R2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &R
                          let r2 = r %s &R{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R1: I {}

                          resource R2: I {}

                          let x <- create R1()
                          let r = &x as &R1
                          let r2 = r %s &R2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyResource -> conforming restricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          resource R: RI {}

                          let x <- create R()
                          let r = &x as &AnyResource{RI}
                          let r2 = r %s &R{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("AnyResource -> conforming restricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          resource R: RI {}

                          let x <- create R()
                          let r = &x as &AnyResource
                          let r2 = r %s &R{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyResource -> non-conforming restricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          resource R {}

                          let x <- create R()
                          let r = &x as &AnyResource{RI}
                          let r2 = r %s &R{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
			})

			// Supertype: Resource (unrestricted)

			t.Run("restricted type -> unrestricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &R{I}
                          let r2 = r %s &R
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          resource T: I {}

                          let x <- create R()
                          let r = &x as &R{I}
                          let t = r %s &T
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyResource -> conforming resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          resource R: RI {}

                          let x <- create R()
                          let r = &x as &AnyResource{RI}
                          let r2 = r %s &R
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyResource -> non-conforming resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          resource R {}

                          let x <- create R()
                          let r = &x as &AnyResource{RI}
                          let r2 = r %s &R
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			})

			t.Run("AnyResource -> unrestricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          resource R: RI {}

                          let x <- create R()
                          let r = &x as &AnyResource
                          let r2 = r %s &R
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			// Supertype: restricted AnyResource

			t.Run("resource -> restricted non-conformance", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          // NOTE: R does not conform to RI
                          resource R {}

                          let x <- create R()
                          let r = &x as &R
                          let r2 = r %s &AnyResource{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("resource -> restricted AnyResource with conformance restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface RI {}

                          resource R: RI {}

                          let x <- create R()
                          let r = &x as &R
                          let r2 = r %s &AnyResource{RI}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted AnyResource with conformance in restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &R{I}
                          let r2 = r %s &AnyResource{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted AnyResource with conformance not in restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &R{I1}
                          let r2 = r %s &AnyResource{I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1 {}

                          let x <- create R()
                          let r = &x as &R{I1}
                          let r2 = r %s &AnyResource{I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

			})

			t.Run("restricted AnyResource -> restricted AnyResource: fewer restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &AnyResource{I1, I2}
                          let r2 = r %s &AnyResource{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: more restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &AnyResource{I1}
                          let r2 = r %s &AnyResource{I1, I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyResource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1 {}

                          let x <- create R()
                          let r = &x as &AnyResource{I1}
                          let r2 = r %s &AnyResource{I1, I2}
		                `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("AnyResource -> restricted AnyResource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &AnyResource
                          let r2 = r %s &AnyResource{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			// Supertype: AnyResource

			t.Run("restricted type -> AnyResource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &R{I1}
                          let r2 = r %s &AnyResource
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted AnyResource -> AnyResource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &AnyResource{I1}
                          let r2 = r %s &AnyResource
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("unrestricted type -> AnyResource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &R
                          let r2 = r %s &AnyResource
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

		})
	}
}

func TestCheckCastUnauthorizedStructReferenceType(t *testing.T) {

	for name, op := range map[string]string{
		"static":  "as",
		"dynamic": "as?",
	} {

		t.Run(name, func(t *testing.T) {

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &S{I1, I2}
                          let s2 = s %s &S{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &S{I1}
                          let s2 = s %s &S{I1, I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S1: I {}

                          struct S2: I {}

                          let x = S1()
                          let s = &x as &S1{I}
                          let s2 = s %s &S2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &S
                          let s2 = s %s &S{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S1: I {}

                          struct S2: I {}

                          let x = S1()
                          let s = &x as &S1
                          let s2 = s %s &S2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyStruct -> conforming restricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          struct S: RI {}

                          let x = S()
                          let s = &x as &AnyStruct{RI}
                          let s2 = s %s &S{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("AnyStruct -> conforming restricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          struct S: RI {}

                          let x = S()
                          let s = &x as &AnyStruct
                          let s2 = s %s &S{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyStruct -> non-conforming restricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          struct S {}

                          let x = S()
                          let s = &x as &AnyStruct{RI}
                          let s2 = s %s &S{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
			})

			// Supertype: Resource (unrestricted)

			t.Run("restricted type -> unrestricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &S{I}
                          let s2 = s %s &S
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          struct T: I {}

                          let x = S()
                          let s = &x as &S{I}
                          let t = s %s &T
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyStruct -> conforming resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          struct S: RI {}

                          let x = S()
                          let s = &x as &AnyStruct{RI}
                          let s2 = s %s &S
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyStruct -> non-conforming resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          struct S {}

                          let x = S()
                          let s = &x as &AnyStruct{RI}
                          let s2 = s %s &S
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			})

			t.Run("AnyStruct -> unrestricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          struct S: RI {}

                          let x = S()
                          let s = &x as &AnyStruct
                          let s2 = s %s &S
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			// Supertype: restricted AnyStruct

			t.Run("resource -> restricted non-conformance", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          // NOTE: R does not conform to RI
                          struct S {}

                          let x = S()
                          let s = &x as &S
                          let s2 = s %s &AnyStruct{RI}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("resource -> restricted AnyStruct with conformance restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface RI {}

                          struct S: RI {}

                          let x = S()
                          let s = &x as &S
                          let s2 = s %s &AnyStruct{RI}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted AnyStruct with conformance in restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &S{I}
                          let s2 = s %s &AnyStruct{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted AnyStruct with conformance not in restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &S{I1}
                          let s2 = s %s &AnyStruct{I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1 {}

                          let x = S()
                          let s = &x as &S{I1}
                          let s2 = s %s &AnyStruct{I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: fewer restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &AnyStruct{I1, I2}
                          let s2 = s %s &AnyStruct{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: more restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &AnyStruct{I1}
                          let s2 = s %s &AnyStruct{I1, I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1 {}

                          let x = S()
                          let s = &x as &AnyStruct{I1}
                          let s2 = s %s &AnyStruct{I1, I2}
		                `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("AnyStruct -> restricted AnyStruct", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &AnyStruct
                          let s2 = s %s &AnyStruct{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			// Supertype: AnyStruct

			t.Run("restricted type -> AnyStruct", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &S{I1}
                          let s2 = s %s &AnyStruct
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted AnyStruct -> AnyStruct", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &AnyStruct{I1}
                          let s2 = s %s &AnyStruct
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("unrestricted type -> AnyStruct", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &S
                          let s2 = s %s &AnyStruct
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})
		})
	}
}

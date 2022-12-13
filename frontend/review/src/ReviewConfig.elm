module ReviewConfig exposing (config)

{-| Do not rename the ReviewConfig module or the config function, because
`elm-review` will look for these.

To add packages that contain rules, add them to this review project using

    `elm install author/packagename`

when inside the directory containing this file.

-}

--import NoMissingTypeAnnotationInLetIn
-- import NoImportingEverything
-- import Docs.ReviewAtDocs
-- the following, we may be able to turn back on once we upgrade elm-spa to v6
-- import NoUnused.Parameters
--import NoUnused.Variables

import NoConfusingPrefixOperator
import NoDebug.Log
import NoDebug.TodoOrToString
import NoExposingEverything
import NoMissingTypeAnnotation
import NoMissingTypeExpose
import NoPrematureLetComputation
import NoSimpleLetBody
import NoUnused.CustomTypeConstructorArgs
import NoUnused.CustomTypeConstructors
import NoUnused.Dependencies
import NoUnused.Exports
import NoUnused.Patterns
import Review.Rule as Rule exposing (Rule)
import Simplify


config : List Rule
config =
    [ -- Docs.ReviewAtDocs.rule
      NoConfusingPrefixOperator.rule
    , NoDebug.Log.rule
    , NoDebug.TodoOrToString.rule
        |> Rule.ignoreErrorsForDirectories [ "tests/" ]
    , NoExposingEverything.rule

    -- , NoImportingEverything.rule []
    , NoMissingTypeAnnotation.rule

    --, NoMissingTypeAnnotationInLetIn.rule
    , NoMissingTypeExpose.rule
    , NoSimpleLetBody.rule
    , NoPrematureLetComputation.rule
    , NoUnused.CustomTypeConstructors.rule []
    , NoUnused.CustomTypeConstructorArgs.rule
    , NoUnused.Dependencies.rule
    , NoUnused.Exports.rule

    --, NoUnused.Parameters.rule
    , NoUnused.Patterns.rule

    --, NoUnused.Variables.rule
    , Simplify.rule Simplify.defaults
    ]

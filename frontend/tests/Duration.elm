module Duration exposing (all)

import Data.Duration as Duration
import Expect
import Test exposing (..)


all : Test
all =
    describe "The Duration module"
        [ test "toString" <|
            \_ ->
                Expect.equal (Duration.toString 1234123232) "14d 6h 48m 43s"
        ]

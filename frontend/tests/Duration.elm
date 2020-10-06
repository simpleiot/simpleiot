module Duration exposing (all)

import Expect
import Test exposing (..)
import Utils.Duration as Duration


all : Test
all =
    describe "The Duration module"
        [ test "toString" <|
            \_ ->
                Expect.equal (Duration.toString 1234123232) "14d 6h 48m 43s"
        , test "toString s only" <|
            \_ ->
                Expect.equal (Duration.toString <| 45 * 1000) "45s"
        , test "toString 1s over a day" <|
            \_ ->
                Expect.equal (Duration.toString <| 1000 * 60 * 60 * 24 + 1000) "1d 1s"
        ]

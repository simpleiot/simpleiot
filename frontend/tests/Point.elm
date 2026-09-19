module Point exposing (all)

import Api.Point as Point exposing (Point)
import Expect
import Test exposing (..)
import Time


all : Test
all =
    describe "Point tests"
        [ test "getTextArray" <|
            \_ ->
                let
                    tzero =
                        Time.millisToPosix 0

                    pts =
                        [ Point "a" "0" tzero 3 0 "111" 0
                        , Point "b" "0" tzero 3 0 "444" 0
                        , Point "a" "2" tzero 3 0 "333" 0
                        , Point "a" "1" tzero 3 0 "222" 0
                        , Point "a" "3" tzero 3 0 "555" 1
                        , Point "a" "10" tzero 3 0 "444" 0
                        ]
                in
                Expect.equal (Point.getTextArray pts "a") [ "111", "222", "333", "444" ]
        , test "mergePoints keeps the newer point" <|
            \_ ->
                let
                    created =
                        Point "tombstone" "0" (Time.millisToPosix 1000) 1 0 "" 0

                    deleted =
                        Point "tombstone" "0" (Time.millisToPosix 2000) 1 1 "" 0
                in
                Point.mergePoints (Point.mergePoints [ created ] [ deleted ]) [ created ]
                    |> Expect.equal [ deleted ]
        ]

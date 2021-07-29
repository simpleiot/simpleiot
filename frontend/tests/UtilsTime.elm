module UtilsTime exposing (local, utc)

import Expect
import Test exposing (..)
import Utils.Time exposing (toLocal, toUTC)


local : Test
local =
    describe "test toLocal"
        [ test "5:00" <|
            \_ ->
                Expect.equal (toLocal -240 "5:00") "01:00"
        , test "1:00" <|
            \_ ->
                Expect.equal (toLocal -240 "1:00") "21:00"
        , test "5:00+4" <|
            \_ ->
                Expect.equal (toLocal 240 "5:00") "09:00"
        , test "1:00+4" <|
            \_ ->
                Expect.equal (toLocal 240 "1:00") "05:00"
        ]


utc : Test
utc =
    describe "test toUTC"
        [ test "5:00" <|
            \_ ->
                Expect.equal (toUTC -240 "1:00") "05:00"
        , test "22:00" <|
            \_ ->
                Expect.equal (toUTC -240 "22:00") "02:00"
        , test "5:00+4" <|
            \_ ->
                Expect.equal (toUTC 240 "1:00") "21:00"
        , test "22:00+4" <|
            \_ ->
                Expect.equal (toUTC 240 "22:00") "18:00"
        ]

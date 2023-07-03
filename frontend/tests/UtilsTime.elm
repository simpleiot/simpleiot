module UtilsTime exposing (local, schedule, utc)

import Expect
import Test exposing (..)
import Utils.Time exposing (scheduleToLocal, scheduleToUTC, toLocal, toUTC)


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


schedule : Test
schedule =
    describe "schedule tests"
        [ test "toLocal no weekday change" <|
            \_ ->
                let
                    sUTC =
                        { startTime = "05:00"
                        , endTime = "08:00"
                        , weekdays = [ 2, 3 ]
                        , dates = []
                        , dateCount = 0
                        }

                    sExp =
                        { startTime = "01:00"
                        , endTime = "04:00"
                        , weekdays = [ 2, 3 ]
                        , dates = []
                        , dateCount = 0
                        }
                in
                Expect.equal sExp <| scheduleToLocal -240 sUTC
        , test "toLocal with weekday change" <|
            \_ ->
                let
                    sUTC =
                        { startTime = "02:00"
                        , endTime = "08:00"
                        , weekdays = [ 0, 2, 3 ]
                        , dates = []
                        , dateCount = 0
                        }

                    sExp =
                        { startTime = "22:00"
                        , endTime = "04:00"
                        , weekdays = [ 1, 2, 6 ]
                        , dates = []
                        , dateCount = 0
                        }
                in
                Expect.equal sExp <| scheduleToLocal -240 sUTC
        , test "toLocal with weekday change pos offset" <|
            \_ ->
                let
                    sUTC =
                        { startTime = "22:00"
                        , endTime = "02:00"
                        , weekdays = [ 2, 3, 6 ]
                        , dates = []
                        , dateCount = 0
                        }

                    sExp =
                        { startTime = "02:00"
                        , endTime = "06:00"
                        , weekdays = [ 0, 3, 4 ]
                        , dates = []
                        , dateCount = 0
                        }
                in
                Expect.equal sExp <| scheduleToLocal 240 sUTC
        , test "toUTC with weekday change" <|
            \_ ->
                let
                    sLocal =
                        { startTime = "22:00"
                        , endTime = "02:00"
                        , weekdays = [ 1, 2, 6 ]
                        , dates = []
                        , dateCount = 0
                        }

                    sExp =
                        { startTime = "02:00"
                        , endTime = "06:00"
                        , weekdays = [ 0, 2, 3 ]
                        , dates = []
                        , dateCount = 0
                        }
                in
                Expect.equal sExp <| scheduleToUTC -240 sLocal
        , test "toUTC with weekday change pos offset" <|
            \_ ->
                let
                    sLocal =
                        { startTime = "02:00"
                        , endTime = "06:00"
                        , weekdays = [ 0, 3, 4 ]
                        , dates = []
                        , dateCount = 0
                        }

                    sExp =
                        { startTime = "22:00"
                        , endTime = "02:00"
                        , weekdays = [ 2, 3, 6 ]
                        , dates = []
                        , dateCount = 0
                        }
                in
                Expect.equal sExp <| scheduleToUTC 240 sLocal
        ]

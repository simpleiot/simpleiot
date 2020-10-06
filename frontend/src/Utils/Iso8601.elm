module Utils.Iso8601 exposing
    ( toUtcString
    , toUtcDateString, toUtcTimeString, toUtcTimeMilliString
    , toUtcDateTimeString, toUtcDateTimeMilliString
    , toString
    , toDateString, toTimeString, toTimeMilliString
    , toDateTimeString, toDateTimeMilliString
    , Mode(..)
    , toTuple
    )

{-| Format a posix time to a ISO8601 String.

None of the generated Strings include timezone postfix.


# UTC strings

@docs toUtcString

@docs toUtcDateString, toUtcTimeString, toUtcTimeMilliString
@docs toUtcDateTimeString, toUtcDateTimeMilliString


# Custom timezone

@docs toString

@docs toDateString, toTimeString, toTimeMilliString
@docs toDateTimeString, toDateTimeMilliString


## Mode for different precission

@docs Mode


# Custom format helper

@docs toTuple

-}

-- this module is copied from https://github.com/f0i/iso8601/blob/1.1.1/src/Iso8601.elm
-- it has name conflicts with Richard F's module that exports the same namespace.
-- BSD 3-Clause license

import Time exposing (Month, Posix, Zone)


{-| The precission of the resulting String can be defined by the Mode parameter

The resulting string will have the following format

  - Year: "YYYY"
  - Month: "YYYY-MM"
  - Day: "YYYY-MM-DD"
  - Hour: "YYYY-MM-DDThh"
  - Minute: "YYYY-MM-DDThh:mm"
  - Second: "YYYY-MM-DDThh:mm:ss"
  - Milli: "YYYY-MM-DDThh:mm:ss.sss"
  - HourOnly: "hh"
  - HourMinute: "hh:mm"
  - HourSecond: "hh:mm:ss"
  - HourMilli: "hh:mm:ss.sss"

-}
type Mode
    = Year
    | Month
    | Day
    | Hour
    | Minute
    | Second
    | Milli
    | HourOnly
    | HourMinute
    | HourSecond
    | HourMilli


{-| Custom seperators
-}
type alias Seperator =
    { date : String
    , time : String
    , dateTime : String
    , millis : String
    }


{-| Format a date ("YYYY-MM-DD")
-}
toDateString : Zone -> Posix -> String
toDateString zone time =
    time |> toString Day zone


{-| Format a time ("hh:mm:ss")
-}
toTimeString : Zone -> Posix -> String
toTimeString zone time =
    time |> toString HourSecond zone


{-| Format a time including millis ("hh:mm:ss.sss")
-}
toTimeMilliString : Zone -> Posix -> String
toTimeMilliString zone time =
    time |> toString HourMilli zone


{-| Format a date time ("YYYY-MM-DDThh:mm:ss")
-}
toDateTimeString : Zone -> Posix -> String
toDateTimeString zone time =
    time |> toString Second zone


{-| Format a time including millis ("YYYY-MM-DDThh:mm:ss.sss")
-}
toDateTimeMilliString : Zone -> Posix -> String
toDateTimeMilliString zone time =
    time |> toString Milli zone


{-| Format a String without timezone offset
-}
toUtcString : Mode -> Posix -> String
toUtcString mode time =
    toString mode Time.utc time


{-| Format a date ("YYYY-MM-DD")
-}
toUtcDateString : Posix -> String
toUtcDateString time =
    time |> toString Day Time.utc


{-| Format a time ("hh:mm:ss")
-}
toUtcTimeString : Posix -> String
toUtcTimeString time =
    time |> toString HourSecond Time.utc


{-| Format a time including millis ("hh:mm:ss.sss")
-}
toUtcTimeMilliString : Posix -> String
toUtcTimeMilliString time =
    time |> toString HourMilli Time.utc


{-| Format a date time ("YYYY-MM-DDThh:mm:ss")
-}
toUtcDateTimeString : Posix -> String
toUtcDateTimeString time =
    time |> toString Second Time.utc


{-| Format a time including millis ("YYYY-MM-DDThh:mm:ss.sss")
-}
toUtcDateTimeMilliString : Posix -> String
toUtcDateTimeMilliString time =
    time |> toString Milli Time.utc


{-| convert a positive integer into a string of at least two digits
-}
iToS2 : Int -> String
iToS2 i =
    if i < 10 then
        "0" ++ String.fromInt i

    else
        String.fromInt i


{-| convert a positive integer into a string of at least three digits
-}
iToS3 : Int -> String
iToS3 i =
    if i < 10 then
        "00" ++ String.fromInt i

    else if i < 100 then
        "0" ++ String.fromInt i

    else
        String.fromInt i


{-| Convert a Time.Month to a number string
-}
monthToS : Month -> String
monthToS month =
    case month of
        Time.Jan ->
            "01"

        Time.Feb ->
            "02"

        Time.Mar ->
            "03"

        Time.Apr ->
            "04"

        Time.May ->
            "05"

        Time.Jun ->
            "06"

        Time.Jul ->
            "07"

        Time.Aug ->
            "08"

        Time.Sep ->
            "09"

        Time.Oct ->
            "10"

        Time.Nov ->
            "11"

        Time.Dec ->
            "12"


{-| Convert a posix time into a ISO8601 string
-}
toString : Mode -> Zone -> Posix -> String
toString mode zone time =
    case mode of
        Year ->
            time |> Time.toYear zone |> String.fromInt

        Month ->
            (time |> Time.toYear zone |> String.fromInt)
                ++ "-"
                ++ (time |> Time.toMonth zone |> monthToS)

        Day ->
            (time |> Time.toYear zone |> String.fromInt)
                ++ "-"
                ++ (time |> Time.toMonth zone |> monthToS)
                ++ "-"
                ++ (time |> Time.toDay zone |> iToS2)

        HourOnly ->
            time |> Time.toHour zone |> iToS2

        HourMinute ->
            (time |> Time.toHour zone |> iToS2)
                ++ ":"
                ++ (time |> Time.toMinute zone |> iToS2)

        HourSecond ->
            (time |> Time.toHour zone |> iToS2)
                ++ ":"
                ++ (time |> Time.toMinute zone |> iToS2)
                ++ ":"
                ++ (time |> Time.toSecond zone |> iToS2)

        HourMilli ->
            (time |> Time.toHour zone |> iToS2)
                ++ ":"
                ++ (time |> Time.toMinute zone |> iToS2)
                ++ ":"
                ++ (time |> Time.toSecond zone |> iToS2)
                ++ "."
                ++ (time |> Time.toMillis zone |> iToS3)

        Hour ->
            (time |> toString Day zone)
                ++ "T"
                ++ (time |> toString HourOnly zone)

        Minute ->
            (time |> toString Day zone)
                ++ "T"
                ++ (time |> toString HourMinute zone)

        Second ->
            (time |> toString Day zone)
                ++ "T"
                ++ (time |> toString HourSecond zone)

        Milli ->
            (time |> toString Day zone)
                ++ "T"
                ++ (time |> toString HourMilli zone)


{-| Get a tuple containg strings for year, month and date
-}
toDateTuple : Zone -> Posix -> ( String, String, String )
toDateTuple zone time =
    let
        yyyy =
            time |> Time.toYear zone |> String.fromInt

        mm =
            time |> Time.toMonth zone |> monthToS

        dd =
            time |> Time.toDay zone |> iToS2
    in
    ( yyyy, mm, dd )


{-| Get a tuple containg strings for hour minute and second
-}
toTimeTuple : Zone -> Posix -> ( String, String, String )
toTimeTuple zone time =
    let
        hh =
            time |> Time.toHour zone |> iToS2

        mm =
            time |> Time.toMinute zone |> iToS2

        ss =
            time |> Time.toSecond zone |> iToS2
    in
    ( hh, mm, ss )


{-| Get a tuple containg strings for all date string parts

This can be used to quickly imlement your own format:

    timeToString time =
        let
            ( ( year, month, day ), ( hour, minute, second ), ms ) =
                toTuple time
        in
        year ++ "/" ++ month ++ "/" ++ day ++ " " ++ hour ++ "_" ++ minute ++ "!"

-}
toTuple : Zone -> Posix -> ( ( String, String, String ), ( String, String, String ), String )
toTuple zone time =
    let
        ms =
            time |> Time.toMillis zone |> iToS3
    in
    ( toDateTuple zone time, toTimeTuple zone time, ms )

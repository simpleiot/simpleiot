module Utils.Iso8601 exposing
    ( toDateTimeString
    , Mode(..)
    , toString
    )

{-| Format a posix time to a ISO8601 String.

None of the generated Strings include timezone postfix.


# UTC strings


# Custom timezone

@docs toDateTimeString


## Mode for different precission

@docs Mode


# Custom format helper

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
    = Day
    | Second
    | HourSecond


{-| Format a date time ("YYYY-MM-DDThh:mm:ss")
-}
toDateTimeString : Zone -> Posix -> String
toDateTimeString zone time =
    time |> toString Second zone


{-| convert a positive integer into a string of at least two digits
-}
iToS2 : Int -> String
iToS2 i =
    if i < 10 then
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
        Day ->
            (time |> Time.toYear zone |> String.fromInt)
                ++ "-"
                ++ (time |> Time.toMonth zone |> monthToS)
                ++ "-"
                ++ (time |> Time.toDay zone |> iToS2)

        HourSecond ->
            (time |> Time.toHour zone |> iToS2)
                ++ ":"
                ++ (time |> Time.toMinute zone |> iToS2)
                ++ ":"
                ++ (time |> Time.toSecond zone |> iToS2)

        Second ->
            (time |> toString Day zone)
                ++ "T"
                ++ (time |> toString HourSecond zone)

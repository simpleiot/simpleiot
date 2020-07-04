module Data.Duration exposing (Duration, toString)


type alias Duration =
    Int


msInSec =
    1000


msInMin =
    msInSec * 60


msInHour =
    msInMin * 60


msInDay =
    msInHour * 24



-- returns days and remainding ms


extract : Duration -> ( Int, Int )
extract dur =
    ( dur // msInDay, modBy msInDay dur )


toString : Duration -> String
toString dur =
    let
        days =
            dur // msInDay

        daysR =
            remainderBy msInDay dur

        hours =
            daysR // msInHour

        hoursR =
            remainderBy msInHour daysR

        minutes =
            hoursR // msInMin

        minutesR =
            remainderBy msInMin hoursR

        seconds =
            minutesR // msInSec
    in
    String.fromInt days
        ++ "d "
        ++ String.fromInt hours
        ++ "h "
        ++ String.fromInt minutes
        ++ "m "
        ++ String.fromInt seconds
        ++ "s"

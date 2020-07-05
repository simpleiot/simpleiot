module Data.Duration exposing (Duration, toString)


type alias Duration =
    Int


msInSec : Int
msInSec =
    1000


msInMin : Int
msInMin =
    msInSec * 60


msInHour : Int
msInHour =
    msInMin * 60


msInDay : Int
msInDay =
    msInHour * 24



-- returns days and remainding ms


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
    (if days > 0 then
        String.fromInt days ++ "d "

     else
        ""
    )
        ++ (if hours > 0 then
                String.fromInt hours ++ "h "

            else
                ""
           )
        ++ (if minutes > 0 then
                String.fromInt minutes ++ "m "

            else
                ""
           )
        ++ (if seconds > 0 || (days <= 0 && hours <= 0 && minutes <= 0) then
                String.fromInt seconds ++ "s"

            else
                ""
           )

module Utils.Time exposing (Schedule, scheduleToLocal, scheduleToUTC, toLocal, toUTC)

import TypedTime exposing (TypedTime)


toLocal : Int -> String -> String
toLocal offset t =
    Maybe.withDefault
        (TypedTime.minutes 0)
        (TypedTime.fromString TypedTime.Minutes t)
        |> TypedTime.add (TypedTime.minutes <| toFloat offset)
        |> normalizeTypedTime
        |> TypedTime.toString TypedTime.Minutes


toLocalWkdayOffset : Int -> String -> ( String, Int )
toLocalWkdayOffset offset t =
    let
        tUTC =
            Maybe.withDefault
                (TypedTime.minutes 0)
                (TypedTime.fromString TypedTime.Minutes t)

        tLocal =
            tUTC
                |> TypedTime.add (TypedTime.minutes <| toFloat offset)
                |> normalizeTypedTime

        wkdayOffset =
            if offset > 0 then
                if TypedTime.lt tLocal tUTC then
                    1

                else
                    0

            else if TypedTime.gt tLocal tUTC then
                -1

            else
                0
    in
    ( tLocal
        |> TypedTime.toString TypedTime.Minutes
    , wkdayOffset
    )


toUTC : Int -> String -> String
toUTC offset t =
    Maybe.withDefault
        (TypedTime.minutes 0)
        (TypedTime.fromString TypedTime.Minutes t)
        |> TypedTime.add (TypedTime.minutes <| negate <| toFloat offset)
        |> normalizeTypedTime
        |> TypedTime.toString TypedTime.Minutes


normalizeTypedTime : TypedTime -> TypedTime
normalizeTypedTime t =
    if TypedTime.lt t (TypedTime.hours 0) then
        TypedTime.add t (TypedTime.hours 24)

    else if TypedTime.gte t (TypedTime.hours 24) then
        TypedTime.sub t (TypedTime.hours 24)

    else
        t


type alias Schedule =
    { startTime : String
    , endTime : String
    , weekdays : List Int
    , dates : List String
    , dateCount : Int
    }


scheduleToLocal : Int -> Schedule -> Schedule
scheduleToLocal offset s =
    let
        ( startTime, wkoff ) =
            toLocalWkdayOffset offset s.startTime

        weekdays =
            List.map (applyWkdayOffset wkoff) s.weekdays |> List.sort

        -- TODO: translate dates
        dates =
            s.dates
    in
    { startTime = startTime
    , endTime = toLocal offset s.endTime
    , weekdays = weekdays
    , dates = dates
    , dateCount = s.dateCount
    }


scheduleToUTC : Int -> Schedule -> Schedule
scheduleToUTC offset s =
    let
        ( startTime, wkoff ) =
            toLocalWkdayOffset (negate offset) s.startTime

        weekdays =
            List.map (applyWkdayOffset wkoff) s.weekdays |> List.sort

        dates =
            List.map (applyDateOffset wkoff) s.dates
    in
    { startTime = startTime
    , endTime = toUTC offset s.endTime
    , weekdays = weekdays
    , dates = dates
    , dateCount = s.dateCount
    }


applyDateOffset : Int -> String -> String
applyDateOffset _ date =
    date


applyWkdayOffset : Int -> Int -> Int
applyWkdayOffset off wkday =
    let
        new =
            wkday + off
    in
    if new < 0 then
        6

    else if new > 6 then
        0

    else
        new

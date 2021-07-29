module Utils.Time exposing (toLocal, toUTC)

import TypedTime exposing (TypedTime)


toLocal : Int -> String -> String
toLocal offset t =
    Maybe.withDefault
        (TypedTime.minutes 0)
        (TypedTime.fromString TypedTime.Minutes t)
        |> TypedTime.add (TypedTime.minutes <| toFloat offset)
        |> normalizeTypedTime
        |> TypedTime.toString TypedTime.Minutes


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

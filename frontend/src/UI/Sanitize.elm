module UI.Sanitize exposing (date, float, hmParser, parseDate, parseHM, time)

import Parser exposing ((|.), Parser)


float : String -> String
float input =
    Tuple.second <| floatHelper ( input, "" )


floatHelper : ( String, String ) -> ( String, String )
floatHelper state =
    let
        sIn =
            Tuple.first state

        sInList =
            String.toList sIn
    in
    case sInList of
        c :: rest ->
            let
                sOut =
                    Tuple.second state
            in
            if (String.length sOut == 0 && c == '-') || Char.isDigit c || (c == '.' && not (String.contains "." sOut)) then
                let
                    sOutList =
                        String.toList sOut
                in
                floatHelper ( String.fromList rest, String.fromList <| sOutList ++ [ c ] )

            else
                state

        _ ->
            state


{-| sanitizeTime looks for a valid time input in the form of hh:mm
this function simply stops when it hits an invalid
portion and returns the valid portion
-}
time : String -> String
time t =
    Tuple.second <| timeHelper ( t, "" )


{-| -- the first value in tuple is incoming string, and 2nd is output
we chomp characters from the input and then process them to the output
-}
timeHelper : ( String, String ) -> ( String, String )
timeHelper state =
    let
        sIn =
            Tuple.first state

        sInList =
            String.toList sIn
    in
    case sInList of
        c :: rest ->
            let
                sOut =
                    Tuple.second state

                sOutList =
                    String.toList sOut

                checkDigit =
                    \ch chRest ->
                        if Char.isDigit ch then
                            timeHelper ( String.fromList chRest, String.fromList <| sOutList ++ [ ch ] )

                        else
                            ( "", sOut )
            in
            case List.length sOutList of
                0 ->
                    checkDigit c rest

                1 ->
                    if c == ':' then
                        -- add a leading digit and try again
                        timeHelper ( sIn, String.fromList <| '0' :: sOutList )

                    else if Char.isDigit c then
                        let
                            sOutNew =
                                String.fromList <| sOutList ++ [ c ]
                        in
                        case String.toInt (String.slice 0 2 sOutNew) of
                            Just hr ->
                                if hr > 23 then
                                    ( "", sOut )

                                else
                                    timeHelper ( String.fromList rest, sOutNew )

                            Nothing ->
                                ( "", sOut )

                    else
                        ( "", sOut )

                2 ->
                    if c == ':' then
                        timeHelper ( String.fromList rest, String.fromList <| sOutList ++ [ c ] )

                    else
                        ( "", sOut )

                3 ->
                    checkDigit c rest

                4 ->
                    if Char.isDigit c then
                        let
                            sOutNew =
                                String.fromList <| sOutList ++ [ c ]
                        in
                        case String.toInt (String.slice 3 5 sOutNew) of
                            Just hr ->
                                if hr > 59 then
                                    ( "", sOut )

                                else
                                    timeHelper ( String.fromList rest, sOutNew )

                            Nothing ->
                                ( "", sOut )

                    else
                        ( "", sOut )

                _ ->
                    ( "", sOut )

        _ ->
            -- we are done
            state


parseHM : String -> Maybe String
parseHM t =
    Parser.run hmParser t
        |> Result.toMaybe


hmParser : Parser String
hmParser =
    Parser.getChompedString <|
        Parser.succeed identity
            |. (Parser.oneOf [ Parser.backtrackable altIntParser, Parser.int ]
                    |> Parser.andThen
                        (\v ->
                            if v < 0 || v > 23 then
                                Parser.problem "hour is out of range"

                            else
                                Parser.succeed v
                        )
               )
            |. Parser.symbol ":"
            |. ((Parser.oneOf [ altIntParser, Parser.int ]
                    |> Parser.andThen
                        (\v ->
                            if v < 0 || v > 59 then
                                Parser.problem "minute is not in range"

                            else
                                Parser.succeed v
                        )
                )
                    |> Parser.getChompedString
                    |> Parser.andThen
                        (\s ->
                            if String.length s /= 2 then
                                Parser.problem "minute must be 2 digits"

                            else
                                Parser.succeed s
                        )
               )


parseDate : String -> Maybe String
parseDate d =
    Parser.run dateParser d
        |> Result.toMaybe


dateParser : Parser String
dateParser =
    Parser.getChompedString <|
        Parser.succeed identity
            |. (Parser.oneOf [ Parser.backtrackable altIntParser, Parser.int ]
                    |> Parser.andThen
                        (\v ->
                            if v < 2023 || v > 2099 then
                                Parser.problem "year is out of range"

                            else
                                Parser.succeed v
                        )
               )
            |. Parser.symbol "-"
            |. ((Parser.oneOf [ altIntParser, Parser.int ]
                    |> Parser.andThen
                        (\v ->
                            if v < 1 || v > 12 then
                                Parser.problem "month is not in range"

                            else
                                Parser.succeed v
                        )
                )
                    |> Parser.getChompedString
                    |> Parser.andThen
                        (\s ->
                            if String.length s /= 2 then
                                Parser.problem "month must be 2 digits"

                            else
                                Parser.succeed s
                        )
               )
            |. Parser.symbol "-"
            |. ((Parser.oneOf [ altIntParser, Parser.int ]
                    |> Parser.andThen
                        (\v ->
                            if v < 1 || v > 31 then
                                Parser.problem "day is not in range"

                            else
                                Parser.succeed v
                        )
                )
                    |> Parser.getChompedString
                    |> Parser.andThen
                        (\s ->
                            if String.length s /= 2 then
                                Parser.problem "day must be 2 digits"

                            else
                                Parser.succeed s
                        )
               )


altIntParser : Parser Int
altIntParser =
    Parser.symbol "0" |> Parser.andThen (\_ -> Parser.int)


{-| date looks for a valid date input in the form of YYYY-MM-DD
this function simply stops when it hits an invalid
portion and returns the valid portion
-}
date : String -> String
date d =
    Tuple.second <| dateHelper ( d, "" )


{-| -- the first value in tuple is incoming string, and 2nd is output
we chomp characters from the input and then process them to the output
0123456789
YYYY-MM-DD
-}
dateHelper : ( String, String ) -> ( String, String )
dateHelper state =
    let
        sIn =
            Tuple.first state

        sInList =
            String.toList sIn
    in
    case sInList of
        c :: rest ->
            let
                sOut =
                    Tuple.second state

                sOutList =
                    String.toList sOut

                checkDigit =
                    \ch chRest ->
                        if Char.isDigit ch then
                            dateHelper ( String.fromList chRest, String.fromList <| sOutList ++ [ ch ] )

                        else
                            ( "", sOut )
            in
            case List.length sOutList of
                0 ->
                    if c /= '2' then
                        ( "", sOut )

                    else
                        checkDigit c rest

                1 ->
                    if c /= '0' then
                        ( "", sOut )

                    else
                        checkDigit c rest

                2 ->
                    checkDigit c rest

                3 ->
                    checkDigit c rest

                4 ->
                    if c == '-' then
                        dateHelper ( String.fromList rest, String.fromList <| sOutList ++ [ c ] )

                    else
                        ( "", sOut )

                5 ->
                    checkDigit c rest

                6 ->
                    if c == '-' then
                        dateHelper
                            ( sIn
                            , String.fromList <|
                                List.take 5 sOutList
                                    ++ '0'
                                    :: List.drop 5 sOutList
                            )

                    else if Char.isDigit c then
                        let
                            sOutNew =
                                String.fromList <| sOutList ++ [ c ]
                        in
                        case String.toInt (String.slice 5 7 sOutNew) of
                            Just mo ->
                                if mo > 12 then
                                    ( "", sOut )

                                else
                                    dateHelper ( String.fromList rest, sOutNew )

                            Nothing ->
                                ( "", sOut )

                    else
                        ( "", sOut )

                7 ->
                    if c == '-' then
                        dateHelper ( String.fromList rest, String.fromList <| sOutList ++ [ c ] )

                    else
                        ( "", sOut )

                8 ->
                    checkDigit c rest

                9 ->
                    if Char.isDigit c then
                        let
                            sOutNew =
                                String.fromList <| sOutList ++ [ c ]
                        in
                        case String.toInt (String.slice 8 10 sOutNew) of
                            Just day ->
                                if day > 31 then
                                    ( "", sOut )

                                else
                                    dateHelper ( String.fromList rest, sOutNew )

                            Nothing ->
                                ( "", sOut )

                    else
                        ( "", sOut )

                _ ->
                    ( "", sOut )

        _ ->
            -- we are done
            state

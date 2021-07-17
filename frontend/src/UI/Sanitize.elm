module UI.Sanitize exposing (float, time)


float : String -> String
float input =
    Tuple.second <| floatHelper ( input, "" )


floatHelper : ( String, String ) -> ( String, String )
floatHelper state =
    let
        sIn =
            Tuple.first state

        sOut =
            Tuple.second state

        sInList =
            String.toList sIn

        sOutList =
            String.toList sOut
    in
    case sInList of
        c :: rest ->
            if Char.isDigit c || (c == '.' && not (String.contains "." sOut)) then
                floatHelper ( String.fromList rest, String.fromList <| sOutList ++ [ c ] )

            else
                state

        _ ->
            state



-- sanitizeTime looks for a valid time input in the form of hh:mm
-- this function simply stops when it hits an invalid
-- portion and returns the valid portion


time : String -> String
time t =
    Tuple.second <| timeHelper ( t, "" )


timeHelper : ( String, String ) -> ( String, String )
timeHelper state =
    let
        sIn =
            Tuple.first state

        sOut =
            Tuple.second state

        sInList =
            String.toList sIn

        sOutList =
            String.toList sOut

        checkDigit =
            \c rest ->
                if Char.isDigit c then
                    timeHelper ( String.fromList rest, String.fromList <| sOutList ++ [ c ] )

                else
                    ( "", sOut )
    in
    case sInList of
        c :: rest ->
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

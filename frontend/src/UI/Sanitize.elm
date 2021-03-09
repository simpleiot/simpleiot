module UI.Sanitize exposing (float)


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

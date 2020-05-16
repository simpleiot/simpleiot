module Data.Sample exposing (Sample, decode, encode, renderSample)

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional)
import Json.Encode
import Round


type alias Sample =
    { sType : String
    , id : String
    , value : Float
    }


encode : Sample -> Json.Encode.Value
encode s =
    Json.Encode.object
        [ ( "type", Json.Encode.string <| s.sType )
        , ( "id", Json.Encode.string <| s.id )
        , ( "value", Json.Encode.float <| s.value )
        ]


decode : Decode.Decoder Sample
decode =
    Decode.succeed Sample
        |> optional "type" Decode.string ""
        |> optional "id" Decode.string ""
        |> optional "value" Decode.float 0


renderSample : Sample -> String
renderSample s =
    let
        id =
            if s.id == "" then
                ""

            else
                s.id ++ ": "
    in
    id ++ Round.round 2 s.value ++ " (" ++ s.sType ++ ")"

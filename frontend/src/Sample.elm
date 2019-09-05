module Sample exposing (Sample, encodeSample, renderSample, sampleDecoder)

import Html exposing (Html)
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import Json.Encode
import Round


type alias Sample =
    { sType : String
    , id : String
    , value : Float
    }


encodeSample : Sample -> Json.Encode.Value
encodeSample s =
    Json.Encode.object
        [ ( "type", Json.Encode.string <| s.sType )
        , ( "id", Json.Encode.string <| s.id )
        , ( "value", Json.Encode.float <| s.value )
        ]


sampleDecoder : Decode.Decoder Sample
sampleDecoder =
    Decode.succeed Sample
        |> required "type" Decode.string
        |> optional "id" Decode.string ""
        |> required "value" Decode.float


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

module Sample exposing (Sample, encodeSample, sampleDecoder)

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import Json.Encode


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

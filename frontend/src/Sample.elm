module Sample exposing (Sample, encodeSample, sampleDecoder)

import Json.Decode as Decode
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
    Decode.map3 Sample
        (Decode.field "type" Decode.string)
        (Decode.field "id" Decode.string)
        (Decode.field "value" Decode.float)

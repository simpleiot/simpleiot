module Sample exposing (Sample, encodeSample)

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

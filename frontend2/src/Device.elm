module Device exposing
    ( Config
    , Device
    , decodeList
    )

import Json.Decode as Decode
import Json.Encode as Encode
import Sample exposing (Sample, sampleDecoder)


type alias Device =
    { id : String
    , config : Config
    , state : State
    }


type alias Config =
    { description : String
    }


type alias State =
    { ios : List Sample
    }


decodeList : Decode.Decoder (List Device)
decodeList =
    Decode.list deviceDecoder


deviceDecoder : Decode.Decoder Device
deviceDecoder =
    Decode.map3 Device
        (Decode.field "id" Decode.string)
        (Decode.field "config" deviceConfigDecoder)
        (Decode.field "state" deviceStateDecoder)


samplesDecoder : Decode.Decoder (List Sample)
samplesDecoder =
    Decode.list sampleDecoder


deviceConfigDecoder : Decode.Decoder Config
deviceConfigDecoder =
    Decode.map Config
        (Decode.field "description" Decode.string)


deviceStateDecoder : Decode.Decoder State
deviceStateDecoder =
    Decode.map State
        (Decode.field "ios" samplesDecoder)


deviceConfigEncoder : Config -> Encode.Value
deviceConfigEncoder deviceConfig =
    Encode.object
        [ ( "description", Encode.string deviceConfig.description ) ]

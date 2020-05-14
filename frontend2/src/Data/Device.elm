module Data.Device exposing
    ( Config
    , Device
    , decodeList
    , deviceConfigEncoder
    )

--import Json.Encode as Encode

import Data.Sample exposing (Sample, sampleDecoder)
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import Json.Encode as Encode


type alias Device =
    { id : String
    , config : Config
    , state : State
    , orgs : List String
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
    Decode.succeed Device
        |> required "id" Decode.string
        |> required "config" deviceConfigDecoder
        |> required "state" deviceStateDecoder
        |> optional "orgs" (Decode.list Decode.string) []


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



--deviceConfigEncoder : Config -> Encode.Value
--deviceConfigEncoder deviceConfig =
--    Encode.object
--        [ ( "description", Encode.string deviceConfig.description ) ]

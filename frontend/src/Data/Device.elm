module Data.Device exposing
    ( Config
    , Device
    , decode
    , decodeList
    , encodeConfig
    , encodeGroups
    )

--import Json.Encode as Encode

import Data.Sample as Sample
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import Json.Encode as Encode


type alias Device =
    { id : String
    , config : Config
    , state : State
    , groups : List String
    }


type alias Config =
    { description : String
    }


type alias State =
    { ios : List Sample.Sample
    }


decodeList : Decode.Decoder (List Device)
decodeList =
    Decode.list decode


decode : Decode.Decoder Device
decode =
    Decode.succeed Device
        |> required "id" Decode.string
        |> required "config" decodeConfig
        |> required "state" decodeState
        |> optional "groups" (Decode.list Decode.string) []


decodeConfig : Decode.Decoder Config
decodeConfig =
    Decode.map Config
        (Decode.field "description" Decode.string)


decodeState : Decode.Decoder State
decodeState =
    Decode.map State
        (Decode.field "ios" (Decode.list Sample.decode))


encodeConfig : Config -> Encode.Value
encodeConfig deviceConfig =
    Encode.object
        [ ( "description", Encode.string deviceConfig.description ) ]


encodeGroups : List String -> Encode.Value
encodeGroups groups =
    Encode.list Encode.string groups

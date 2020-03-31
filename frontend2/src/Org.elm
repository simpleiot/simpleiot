module Org exposing
    ( Org
    , decodeList
    )

import Device
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import User


type alias Org =
    { id : String
    , name : String
    , users : List User.User
    , devices : List Device.Device
    }


decodeList : Decode.Decoder (List Org)
decodeList =
    Decode.list decode


decode =
    Decode.succeed Org
        |> required "id" Decode.string
        |> required "name" Decode.string
        |> required "users" User.decodeList
        |> required "devices" Device.decodeList

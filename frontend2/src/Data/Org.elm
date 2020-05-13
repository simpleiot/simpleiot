module Data.Org exposing
    ( Org
    , decodeList
    )

import Data.Device as D
import Data.User as U
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)


type alias Org =
    { id : String
    , name : String
    , users : List U.User
    , devices : List D.Device
    }


decodeList : Decode.Decoder (List Org)
decodeList =
    Decode.list decode


decode : Decode.Decoder Org
decode =
    Decode.succeed Org
        |> required "id" Decode.string
        |> required "name" Decode.string
        |> required "users" U.decodeList
        |> required "devices" D.decodeList

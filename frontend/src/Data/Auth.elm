module Data.Auth exposing (..)

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)


type alias Cred =
    { email : String
    , password : String
    }


type alias Response =
    { token : String
    , isRoot : Bool
    }


decodeResponse : Decode.Decoder Response
decodeResponse =
    Decode.succeed Response
        |> required "token" Decode.string
        |> required "isRoot" Decode.bool

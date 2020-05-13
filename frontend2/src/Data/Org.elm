module Data.Org exposing
    ( Org
    , decodeList
    )

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)


type alias Org =
    { id : String
    , name : String
    , parent : String
    , users : List UserRoles
    }


type alias UserRoles =
    { userId : String
    , roles : List String
    }


decodeUserRoles : Decode.Decoder UserRoles
decodeUserRoles =
    Decode.succeed UserRoles
        |> required "userId" Decode.string
        |> required "roles" (Decode.list Decode.string)


decodeList : Decode.Decoder (List Org)
decodeList =
    Decode.list decode


decode : Decode.Decoder Org
decode =
    Decode.succeed Org
        |> required "id" Decode.string
        |> required "name" Decode.string
        |> required "parent" Decode.string
        |> required "users" (Decode.list decodeUserRoles)

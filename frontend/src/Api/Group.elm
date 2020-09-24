module Data.Group exposing
    ( Group
    , UserRoles
    , decodeList
    , empty
    , encode
    )

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import Json.Encode as Encode


type alias Group =
    { id : String
    , name : String
    , parent : String
    , users : List UserRoles
    }


empty : Group
empty =
    { id = ""
    , name = ""
    , parent = ""
    , users = []
    }


type alias UserRoles =
    { userId : String
    , roles : List String
    }


decodeList : Decode.Decoder (List Group)
decodeList =
    Decode.list decode


decode : Decode.Decoder Group
decode =
    Decode.succeed Group
        |> required "id" Decode.string
        |> required "name" Decode.string
        |> required "parent" Decode.string
        |> optional "users" (Decode.list decodeUserRoles) []


decodeUserRoles : Decode.Decoder UserRoles
decodeUserRoles =
    Decode.succeed UserRoles
        |> required "userId" Decode.string
        |> required "roles" (Decode.list Decode.string)


encode : Group -> Encode.Value
encode group =
    Encode.object
        [ ( "name", Encode.string group.name )
        , ( "users", Encode.list encodeUserRoles group.users )
        ]


encodeUserRoles : UserRoles -> Encode.Value
encodeUserRoles userRoles =
    Encode.object
        [ ( "userId", Encode.string userRoles.userId )
        , ( "roles", Encode.list Encode.string userRoles.roles )
        ]

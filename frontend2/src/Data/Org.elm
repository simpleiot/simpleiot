module Data.Org exposing
    ( Org
    , UserRoles
    , decodeList
    , empty
    , encode
    )

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import Json.Encode as Encode


type alias Org =
    { id : String
    , name : String
    , parent : String
    , users : List UserRoles
    }


empty : Org
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


decodeList : Decode.Decoder (List Org)
decodeList =
    Decode.list decode


decode : Decode.Decoder Org
decode =
    Decode.succeed Org
        |> required "id" Decode.string
        |> required "name" Decode.string
        |> required "parent" Decode.string
        |> optional "users" (Decode.list decodeUserRoles) []


decodeUserRoles : Decode.Decoder UserRoles
decodeUserRoles =
    Decode.succeed UserRoles
        |> required "userId" Decode.string
        |> required "roles" (Decode.list Decode.string)


encode : Org -> Encode.Value
encode org =
    Encode.object
        [ ( "name", Encode.string org.name )
        , ( "users", Encode.list encodeUserRoles org.users )
        ]


encodeUserRoles : UserRoles -> Encode.Value
encodeUserRoles userRoles =
    Encode.object
        [ ( "userId", Encode.string userRoles.userId )
        , ( "roles", Encode.list Encode.string userRoles.roles )
        ]

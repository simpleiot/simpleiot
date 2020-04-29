module User exposing
    ( User
    , decodeList
    , empty
    , encode
    )

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import Json.Encode as Encode


type alias User =
    { id : String
    , first : String
    , last : String
    , email : String
    , pass : String
    , roles : List Role
    }


empty =
    { id = ""
    , email = ""
    , pass = ""
    , first = ""
    , last = ""
    , roles = []
    }


decodeList : Decode.Decoder (List User)
decodeList =
    Decode.list decode


decode =
    Decode.succeed User
        |> required "id" Decode.string
        |> required "firstName" Decode.string
        |> required "lastName" Decode.string
        |> required "email" Decode.string
        |> required "pass" Decode.string
        |> required "roles" decodeRoles


encode : User -> Encode.Value
encode user =
    Encode.object
        [ ( "firstName", Encode.string user.first )
        , ( "lastName", Encode.string user.last )
        , ( "email", Encode.string user.email )
        , ( "pass", Encode.string user.pass )
        , ( "roles", encodeRoles user.roles )
        ]


type alias Role =
    { id : String
    , orgID : String
    , orgName : String
    , description : String
    }


encodeRole role =
    Encode.object
        [ ( "id", Encode.string role.id )
        , ( "orgID", Encode.string role.orgID )
        , ( "orgName", Encode.string role.orgName )
        , ( "description", Encode.string role.description )
        ]


encodeRoles =
    Encode.list encodeRole


decodeRole =
    Decode.succeed Role
        |> required "id" Decode.string
        |> required "orgID" Decode.string
        |> required "orgName" Decode.string
        |> required "description" Decode.string

decodeRoles =
    Decode.list decodeRole

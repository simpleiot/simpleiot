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
    , admin : Bool
    }


empty =
    { id = ""
    , admin = False
    , email = ""
    , first = ""
    , last = ""
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
        |> optional "admin" Decode.bool False


encode : User -> Encode.Value
encode user =
    Encode.object
        [ ( "firstName", Encode.string user.first )
        , ( "lastName", Encode.string user.last )
        , ( "email", Encode.string user.email )
        ]

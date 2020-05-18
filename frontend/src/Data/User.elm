module Data.User exposing
    ( User
    , decode
    , decodeList
    , empty
    , encode
    , findUser
    )

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)
import Json.Encode as Encode
import List.Extra


type alias User =
    { id : String
    , first : String
    , last : String
    , email : String
    , pass : String
    }


empty : User
empty =
    { id = ""
    , email = ""
    , pass = ""
    , first = ""
    , last = ""
    }


decodeList : Decode.Decoder (List User)
decodeList =
    Decode.list decode


decode : Decode.Decoder User
decode =
    Decode.succeed User
        |> required "id" Decode.string
        |> required "firstName" Decode.string
        |> required "lastName" Decode.string
        |> required "email" Decode.string
        |> required "pass" Decode.string


encode : User -> Encode.Value
encode user =
    Encode.object
        [ ( "firstName", Encode.string user.first )
        , ( "lastName", Encode.string user.last )
        , ( "email", Encode.string user.email )
        , ( "pass", Encode.string user.pass )
        ]


findUser : List User -> String -> Maybe User
findUser users id =
    List.Extra.find (\u -> u.id == id) users

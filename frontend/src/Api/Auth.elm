module Api.Auth exposing
    ( User
    , decode
    , encode
    , login
    )

import Api.Data exposing (Data)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)
import Json.Encode as Encode
import Url.Builder


type alias User =
    { token : String
    , email : String
    }


decode : Decode.Decoder User
decode =
    Decode.succeed User
        |> required "token" Decode.string
        |> required "email" Decode.string


encode : User -> Encode.Value
encode user =
    Encode.object
        [ ( "token", Encode.string user.token )
        , ( "email", Encode.string user.email )
        ]


login :
    { user : { user | email : String, password : String }
    , onResponse : Data User -> msg
    }
    -> Cmd msg
login options =
    Http.post
        { body =
            Http.multipartBody
                [ Http.stringPart "email" options.user.email
                , Http.stringPart "password" options.user.password
                ]
        , url = Url.Builder.absolute [ "v1", "auth" ] []
        , expect = Api.Data.expectJson options.onResponse decode
        }

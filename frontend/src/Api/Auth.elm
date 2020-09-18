module Api.Auth exposing
    ( Auth
    , Cred
    , login
    )

import Api.Data exposing (Data)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)
import Url.Builder


type alias Cred =
    { email : String
    , password : String
    }


type alias Auth =
    { token : String
    , isRoot : Bool
    }


decodeResponse : Decode.Decoder Auth
decodeResponse =
    Decode.succeed Auth
        |> required "token" Decode.string
        |> required "isRoot" Decode.bool


login :
    { user : { user | email : String, password : String }
    , onResponse : Data Auth -> msg
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
        , expect = Api.Data.expectJson options.onResponse decodeResponse
        }

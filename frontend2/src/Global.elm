module Global exposing
    ( Flags
    , Model(..)
    , Msg(..)
    , Session
    , init
    , subscriptions
    , update
    )

import Data.Data as Data
import Data.Device as D
import Data.Org as O
import Data.User as U
import Generated.Routes exposing (Route, routes)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import List.Extra
import Time
import Url.Builder as Url


type alias Flags =
    ()


type Model
    = SignedOut (Maybe Http.Error)
    | SignedIn Session


type alias Session =
    { cred : Cred
    , authToken : String
    , data : Data.Data
    , error : Maybe Http.Error
    , respError : Maybe String
    , posting : Bool
    , newOrgUser : Maybe U.User
    , newOrgDevice : Maybe D.Device
    }


type alias Cred =
    { email : String
    , password : String
    }


type Msg
    = SignIn Cred
    | AuthResponse Cred (Result Http.Error Auth)
    | RequestOrgs
    | RequestDevices
    | RequestUsers
    | DevicesResponse (Result Http.Error (List D.Device))
    | OrgsResponse (Result Http.Error (List O.Org))
    | UsersResponse (Result Http.Error (List U.User))
    | DeleteDevice String
    | DeleteDeviceResponse String (Result Http.Error Response)
    | SignOut
    | Tick Time.Posix
    | UpdateDeviceConfig String D.Config
    | UpdateDeviceOrgs String (List String)
    | UpdateUser U.User
    | DeleteUser String
    | DeleteUserResponse String (Result Http.Error Response)
    | UpdateOrg O.Org
    | DeleteOrg String
    | DeleteOrgResponse String (Result Http.Error Response)
    | ConfigPosted String (Result Http.Error Response)
    | UserPosted String (Result Http.Error Response)
    | OrgPosted String (Result Http.Error Response)
    | CheckUser String
    | CheckUserResponse (Result Http.Error U.User)
    | CheckDevice String
    | CheckDeviceResponse (Result Http.Error D.Device)


type alias Commands msg =
    { navigate : Route -> Cmd msg
    }


init : Commands msg -> Flags -> ( Model, Cmd Msg, Cmd msg )
init _ _ =
    ( SignedOut Nothing
    , Cmd.none
    , Cmd.none
    )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ Time.every 10000 Tick
        ]


login : Cred -> Cmd Msg
login cred =
    Http.post
        { body =
            Http.multipartBody
                [ Http.stringPart "email" cred.email
                , Http.stringPart "password" cred.password
                ]
        , url = Url.absolute [ "v1", "auth" ] []
        , expect = Http.expectJson (AuthResponse cred) decodeAuth
        }


type alias Auth =
    { token : String
    }


decodeAuth : Decode.Decoder Auth
decodeAuth =
    Decode.succeed Auth
        |> required "token" Decode.string


update : Commands msg -> Msg -> Model -> ( Model, Cmd Msg, Cmd msg )
update commands msg model =
    case model of
        SignedOut _ ->
            case msg of
                SignIn cred ->
                    ( SignedOut Nothing
                    , login cred
                    , Cmd.none
                    )

                AuthResponse cred (Ok { token }) ->
                    ( SignedIn
                        { authToken = token
                        , cred = cred
                        , data = Data.empty
                        , error = Nothing
                        , respError = Nothing
                        , posting = False
                        , newOrgUser = Nothing
                        , newOrgDevice = Nothing
                        }
                    , Cmd.none
                    , commands.navigate routes.top
                    )

                AuthResponse _ (Err error) ->
                    ( SignedOut (Just error), Cmd.none, Cmd.none )

                _ ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

        SignedIn sess ->
            let
                data =
                    sess.data
            in
            case msg of
                SignIn _ ->
                    ( model, Cmd.none, Cmd.none )

                SignOut ->
                    ( SignedOut Nothing
                    , Cmd.none
                    , commands.navigate routes.top
                    )

                AuthResponse _ (Ok _) ->
                    ( model, Cmd.none, Cmd.none )

                AuthResponse _ (Err err) ->
                    ( SignedOut (Just err)
                    , Cmd.none
                    , commands.navigate routes.signIn
                    )

                DevicesResponse (Ok devices) ->
                    ( SignedIn
                        { sess
                            | data = { data | devices = devices }
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                DevicesResponse (Err _) ->
                    ( SignedIn
                        { sess
                            | respError = Just "Error getting devices"
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                UsersResponse (Ok users) ->
                    ( SignedIn { sess | data = { data | users = users } }
                    , Cmd.none
                    , Cmd.none
                    )

                UsersResponse (Err _) ->
                    ( SignedIn { sess | respError = Just "Error getting users" }
                    , Cmd.none
                    , Cmd.none
                    )

                RequestDevices ->
                    ( model
                    , if sess.posting then
                        Cmd.none

                      else
                        getDevices sess.authToken
                    , Cmd.none
                    )

                RequestUsers ->
                    ( model
                    , getUsers sess.authToken
                    , Cmd.none
                    )

                OrgsResponse (Ok orgs) ->
                    ( SignedIn { sess | data = { data | orgs = orgs } }
                    , Cmd.none
                    , Cmd.none
                    )

                OrgsResponse (Err _) ->
                    ( SignedIn { sess | respError = Just "Error getting orgs" }
                    , Cmd.none
                    , Cmd.none
                    )

                RequestOrgs ->
                    ( model
                    , getOrgs sess.authToken
                    , Cmd.none
                    )

                Tick _ ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

                UpdateDeviceConfig id config ->
                    let
                        devices =
                            List.map
                                (\d ->
                                    if d.id == id then
                                        { d | config = config }

                                    else
                                        d
                                )
                                data.devices
                    in
                    ( SignedIn
                        { sess
                            | data = { data | devices = devices }
                            , posting = True
                        }
                    , postDeviceConfig sess.authToken id config
                    , Cmd.none
                    )

                UpdateDeviceOrgs id orgs ->
                    let
                        devices =
                            List.map
                                (\d ->
                                    if d.id == id then
                                        { d | orgs = orgs }

                                    else
                                        d
                                )
                                data.devices
                    in
                    ( SignedIn
                        { sess
                            | data = { data | devices = devices }
                            , posting = True
                            , newOrgDevice = Nothing
                        }
                    , postDeviceOrgs sess.authToken id orgs
                    , Cmd.none
                    )

                UpdateUser user ->
                    let
                        -- update local model to make UI optimistic
                        updateUser old =
                            if old.id == user.id then
                                user

                            else
                                old

                        users =
                            if user.id == "" then
                                [ user ] ++ sess.data.users

                            else
                                List.map updateUser sess.data.users
                    in
                    ( SignedIn { sess | data = { data | users = users } }
                    , postUser sess.authToken user
                    , Cmd.none
                    )

                UpdateOrg org ->
                    let
                        -- update local model to make UI optimistic
                        updateOrg old =
                            if old.id == org.id then
                                org

                            else
                                old

                        orgs =
                            if org.id == "" then
                                [ org ] ++ sess.data.orgs

                            else
                                List.map updateOrg sess.data.orgs
                    in
                    ( SignedIn
                        { sess
                            | data = { data | orgs = orgs }
                            , newOrgUser = Nothing
                        }
                    , postOrg sess.authToken org
                    , Cmd.none
                    )

                DeleteOrg id ->
                    let
                        orgs =
                            List.filter (\o -> o.id /= id) data.orgs
                    in
                    ( SignedIn { sess | data = { data | orgs = orgs } }
                    , deleteOrg sess.authToken id
                    , Cmd.none
                    )

                DeleteOrgResponse _ (Ok _) ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

                DeleteOrgResponse _ (Err _) ->
                    ( SignedIn
                        { sess
                            | respError = Just "Error deleting org"
                            , posting = False
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                DeleteDevice id ->
                    let
                        devices =
                            List.filter (\d -> d.id /= id) data.devices
                    in
                    ( SignedIn { sess | data = { data | devices = devices } }
                    , deleteDevice sess.authToken id
                    , Cmd.none
                    )

                DeleteDeviceResponse _ (Ok _) ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

                DeleteDeviceResponse _ (Err _) ->
                    ( SignedIn
                        { sess
                            | respError = Just "Error deleting device"
                            , posting = False
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                ConfigPosted _ (Ok _) ->
                    ( SignedIn { sess | posting = False }
                    , Cmd.none
                    , Cmd.none
                    )

                ConfigPosted _ (Err _) ->
                    ( SignedIn
                        { sess
                            | respError = Just "Error saving device config"
                            , posting = False
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                UserPosted _ (Ok resp) ->
                    -- populate the assigned ID in the new user
                    let
                        users =
                            List.map
                                (\u ->
                                    if u.id == "" then
                                        { u | id = resp.id }

                                    else
                                        u
                                )
                                data.users
                    in
                    ( SignedIn { sess | data = { data | users = users } }
                    , Cmd.none
                    , Cmd.none
                    )

                UserPosted _ (Err _) ->
                    -- refresh users as the local users cache is now
                    -- stale
                    ( SignedIn { sess | respError = Just "Error saving user" }
                    , getUsers sess.authToken
                    , Cmd.none
                    )

                OrgPosted _ (Ok resp) ->
                    -- populate the assigned ID in the new org
                    let
                        orgs =
                            List.map
                                (\o ->
                                    if o.id == "" then
                                        { o | id = resp.id }

                                    else
                                        o
                                )
                                data.orgs
                    in
                    ( SignedIn { sess | data = { data | orgs = orgs } }
                    , Cmd.none
                    , Cmd.none
                    )

                OrgPosted _ (Err _) ->
                    -- refresh the ids because the local org cache is
                    -- is not correct because save did not take
                    ( SignedIn { sess | respError = Just "Error saving org" }
                    , getOrgs sess.authToken
                    , Cmd.none
                    )

                CheckUser userEmail ->
                    ( SignedIn { sess | newOrgUser = Nothing }
                    , getUserByEmail sess.authToken userEmail
                    , Cmd.none
                    )

                CheckUserResponse (Err _) ->
                    ( model, Cmd.none, Cmd.none )

                CheckUserResponse (Ok user) ->
                    ( SignedIn { sess | newOrgUser = Just user }
                    , Cmd.none
                    , Cmd.none
                    )

                CheckDevice deviceId ->
                    ( SignedIn { sess | newOrgDevice = Nothing }
                    , getDeviceById sess.authToken deviceId
                    , Cmd.none
                    )

                CheckDeviceResponse (Err _) ->
                    ( model, Cmd.none, Cmd.none )

                CheckDeviceResponse (Ok device) ->
                    -- make sure new device is in our local cache
                    -- of devices so we can modify it if necessary
                    let
                        devices =
                            case
                                List.Extra.find (\d -> d.id == device.id)
                                    sess.data.devices
                            of
                                Just _ ->
                                    sess.data.devices

                                Nothing ->
                                    device :: sess.data.devices
                    in
                    ( SignedIn
                        { sess
                            | newOrgDevice = Just device
                            , data = { data | devices = devices }
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                DeleteUser id ->
                    let
                        users =
                            List.filter (\u -> u.id /= id) data.users
                    in
                    ( SignedIn { sess | data = { data | users = users } }
                    , deleteUser sess.authToken id
                    , Cmd.none
                    )

                DeleteUserResponse _ (Ok _) ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

                DeleteUserResponse _ (Err _) ->
                    ( SignedIn
                        { sess
                            | respError = Just "Error deleting user"
                            , posting = False
                        }
                    , getUsers sess.authToken
                    , Cmd.none
                    )


getDevices : String -> Cmd Msg
getDevices token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices" ] []
        , expect = Http.expectJson DevicesResponse D.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getDeviceById : String -> String -> Cmd Msg
getDeviceById token id =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices", id ] []
        , expect = Http.expectJson CheckDeviceResponse D.decode
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


deleteDevice : String -> String -> Cmd Msg
deleteDevice token id =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices", id ] []
        , expect = Http.expectJson (DeleteDeviceResponse id) responseDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }


responseDecoder : Decode.Decoder Response
responseDecoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""
        |> optional "id" Decode.string ""


postDeviceConfig : String -> String -> D.Config -> Cmd Msg
postDeviceConfig token id config =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices", id, "config" ] []
        , expect = Http.expectJson (ConfigPosted id) responseDecoder
        , body = config |> D.encodeConfig |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postDeviceOrgs : String -> String -> List String -> Cmd Msg
postDeviceOrgs token id orgs =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices", id, "orgs" ] []
        , expect = Http.expectJson (ConfigPosted id) responseDecoder
        , body = orgs |> D.encodeOrgs |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


getOrgs : String -> Cmd Msg
getOrgs token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "orgs" ] []
        , expect = Http.expectJson OrgsResponse O.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getUsers : String -> Cmd Msg
getUsers token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users" ] []
        , expect = Http.expectJson UsersResponse U.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


deleteUser : String -> String -> Cmd Msg
deleteUser token id =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users", id ] []
        , expect = Http.expectJson (DeleteUserResponse id) responseDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getUserByEmail : String -> String -> Cmd Msg
getUserByEmail token email =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users" ] [ Url.string "email" email ]
        , expect = Http.expectJson CheckUserResponse U.decode
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


postUser : String -> U.User -> Cmd Msg
postUser token user =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users", user.id ] []
        , expect = Http.expectJson (UserPosted user.id) responseDecoder
        , body = user |> U.encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postOrg : String -> O.Org -> Cmd Msg
postOrg token org =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "orgs", org.id ] []
        , expect = Http.expectJson (OrgPosted org.id) responseDecoder
        , body = org |> O.encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


deleteOrg : String -> String -> Cmd Msg
deleteOrg token id =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "orgs", id ] []
        , expect = Http.expectJson (DeleteOrgResponse id) responseDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }
